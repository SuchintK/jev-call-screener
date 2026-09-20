package jev

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SuchintK/jev-call-screener/internal/domain"
)

func TestRequestAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("authorization = %q", got)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["model"] != "jev-test" {
			t.Errorf("model = %#v", request["model"])
		}
		state := request["state"].(map[string]any)
		if state["caller_transcript"] != "loan offer" {
			t.Errorf("transcript = %#v", state["caller_transcript"])
		}
		questions := request["questions"].(map[string]any)
		callType := questions["call_type"].(map[string]any)
		if callType["type"] != "choice" {
			t.Errorf("question type = %#v", callType["type"])
		}
		_, _ = w.Write([]byte(`{"model":"jev-response","questions":{"call_type":{"choice":"promotional","confidence":0.97,"probabilities":{"promotional":0.97,"wanted":0.01,"unclear":0.02}}}}`))
	}))
	defer server.Close()

	client := NewClient("secret", "jev-test", time.Second, WithEndpoint(server.URL), WithRetry(0, 0))
	result, err := client.Classify(context.Background(), "loan offer")
	if err != nil {
		t.Fatal(err)
	}
	if result.Category != domain.CategoryPromotional || result.Confidence != .97 || result.Model != "jev-response" || result.Probabilities[domain.CategoryUnclear] != .02 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestHTTPFailures(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusUnprocessableEntity, http.StatusTooManyRequests, 529} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
			}))
			defer server.Close()
			client := NewClient("secret", "jev-test", time.Second, WithEndpoint(server.URL), WithRetry(1, 0))
			_, err := client.Classify(context.Background(), "test")
			var httpErr *HTTPError
			if !errors.As(err, &httpErr) || httpErr.StatusCode != status {
				t.Fatalf("error = %v", err)
			}
			wantCalls := int32(1)
			if status == http.StatusTooManyRequests || status >= 500 {
				wantCalls = 2
			}
			if calls.Load() != wantCalls {
				t.Fatalf("calls = %d, want %d", calls.Load(), wantCalls)
			}
		})
	}
}

func TestMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"questions":{"call_type":{"choice":"wanted"}}}`))
	}))
	defer server.Close()
	client := NewClient("secret", "jev-test", time.Second, WithEndpoint(server.URL), WithRetry(0, 0))
	if _, err := client.Classify(context.Background(), "test"); err == nil {
		t.Fatal("expected malformed response error")
	}
}

func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	client := NewClient("secret", "jev-test", 10*time.Millisecond, WithEndpoint(server.URL), WithRetry(0, 0))
	if _, err := client.Classify(context.Background(), "test"); err == nil {
		t.Fatal("expected timeout error")
	}
}
