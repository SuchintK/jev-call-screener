package twilio

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/SuchintK/jev-call-screener/internal/adapters/memory"
	"github.com/SuchintK/jev-call-screener/internal/adapters/mockclassifier"
	"github.com/SuchintK/jev-call-screener/internal/application"
	"github.com/SuchintK/jev-call-screener/internal/domain"
)

func testService(classification domain.Classification) *application.ScreeningService {
	classifier := mockclassifier.New(mockclassifier.WithFallback(classification))
	return application.NewScreeningService(classifier, application.RoutingPolicy{
		PromotionalRejectThreshold: .90,
		WantedForwardThreshold:     .75,
		MaxClarificationTurns:      1,
		ForwardOnError:             true,
	})
}

func formRequest(path string, values url.Values) *http.Request {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}

func TestIncomingGeneratesGather(t *testing.T) {
	recorder := httptest.NewRecorder()
	IncomingHandler{Sessions: memory.NewSessionStore(), Greeting: "Tell me why"}.ServeHTTP(recorder, formRequest("/webhooks/twilio/incoming", url.Values{"CallSid": {"CA1"}}))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "<Gather") || !strings.Contains(recorder.Body.String(), "Tell me why") {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestGatherActions(t *testing.T) {
	tests := []struct {
		name           string
		classification domain.Classification
		wantXML        string
	}{
		{"promotional hangs up", domain.Classification{Category: domain.CategoryPromotional, Confidence: .97}, "<Hangup"},
		{"wanted dials", domain.Classification{Category: domain.CategoryWanted, Confidence: .90}, "<Dial>+15551234567</Dial>"},
		{"unclear gathers", domain.Classification{Category: domain.CategoryUnclear, Confidence: .80}, "<Gather"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler := GatherHandler{Service: testService(test.classification), Sessions: memory.NewSessionStore(), ForwardToNumber: "+15551234567", ClarificationPrompt: "More detail", RejectedMessage: "No sales"}
			handler.ServeHTTP(recorder, formRequest("/webhooks/twilio/gather", url.Values{"CallSid": {"CA1"}, "SpeechResult": {"hello"}}))
			if !strings.Contains(recorder.Body.String(), test.wantXML) {
				t.Fatalf("response = %s, want %s", recorder.Body.String(), test.wantXML)
			}
		})
	}
}

func TestSecondUnclearTurnForwardsAndCombinesContext(t *testing.T) {
	store := memory.NewSessionStore()
	// A recording classifier checks the exact combined transcript.
	recording := &recordingClassifier{classification: domain.Classification{Category: domain.CategoryUnclear, Confidence: .5}}
	service := application.NewScreeningService(recording, application.RoutingPolicy{PromotionalRejectThreshold: .9, WantedForwardThreshold: .75, MaxClarificationTurns: 1, ForwardOnError: true})
	handler := GatherHandler{Service: service, Sessions: store, ForwardToNumber: "+15551234567", ClarificationPrompt: "More", RejectedMessage: "No"}

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, formRequest("/webhooks/twilio/gather", url.Values{"CallSid": {"CA1"}, "SpeechResult": {"An opportunity"}}))
	if !strings.Contains(first.Body.String(), "<Gather") {
		t.Fatalf("first response = %s", first.Body.String())
	}
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, formRequest("/webhooks/twilio/gather", url.Values{"CallSid": {"CA1"}, "SpeechResult": {"A job interview"}}))
	if !strings.Contains(second.Body.String(), "<Dial>") {
		t.Fatalf("second response = %s", second.Body.String())
	}
	want := "Caller initial response:\nAn opportunity\n\nCaller clarification:\nA job interview"
	if recording.last != want {
		t.Fatalf("combined transcript = %q, want %q", recording.last, want)
	}
}

type recordingClassifier struct {
	classification domain.Classification
	last           string
}

func (c *recordingClassifier) Classify(_ context.Context, transcript string) (domain.Classification, error) {
	c.last = transcript
	return c.classification, nil
}

func TestMissingSpeechResultHandledSafely(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := GatherHandler{Service: testService(domain.Classification{}), Sessions: memory.NewSessionStore(), ForwardToNumber: "+15551234567", ClarificationPrompt: "More detail", RejectedMessage: "No sales"}
	handler.ServeHTTP(recorder, formRequest("/webhooks/twilio/gather", url.Values{"CallSid": {"CA1"}}))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "<Gather") {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestInvalidSignatureRejected(t *testing.T) {
	nextCalled := false
	handler := ValidateSignatures(SignatureValidator{AuthToken: "token", PublicBaseURL: "https://calls.example.com"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true }))
	request := formRequest("/webhooks/twilio/incoming", url.Values{"CallSid": {"CA1"}})
	request.Header.Set("X-Twilio-Signature", "invalid")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden || nextCalled {
		t.Fatalf("status=%d nextCalled=%v", recorder.Code, nextCalled)
	}
}

func TestValidSignatureAccepted(t *testing.T) {
	values := url.Values{"CallSid": {"CA1"}}
	data := "https://calls.example.com/webhooks/twilio/incoming" + "CallSid" + "CA1"
	mac := hmac.New(sha1.New, []byte("token"))
	_, _ = mac.Write([]byte(data))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	handler := ValidateSignatures(SignatureValidator{AuthToken: "token", PublicBaseURL: "https://calls.example.com"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	request := formRequest("/webhooks/twilio/incoming", values)
	request.Header.Set("X-Twilio-Signature", signature)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
}
