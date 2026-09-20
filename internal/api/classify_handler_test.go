package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SuchintK/jev-call-screener/internal/adapters/mockclassifier"
	"github.com/SuchintK/jev-call-screener/internal/application"
	"github.com/SuchintK/jev-call-screener/internal/domain"
)

func TestClassifyEndpoint(t *testing.T) {
	classifier := mockclassifier.New(mockclassifier.WithResponse("loan", domain.Classification{
		Category: domain.CategoryPromotional, Confidence: .97,
		Probabilities: map[domain.Category]float64{domain.CategoryPromotional: .97, domain.CategoryWanted: .01, domain.CategoryUnclear: .02}, Model: "mock",
	}))
	service := application.NewScreeningService(classifier, application.RoutingPolicy{PromotionalRejectThreshold: .9, WantedForwardThreshold: .75, MaxClarificationTurns: 1, ForwardOnError: true})
	router := NewRouter(RouterOptions{Classify: ClassifyHandler{Service: service}})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/classify", strings.NewReader(`{"transcript":"personal loan"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"action":"reject"`) || !strings.Contains(recorder.Body.String(), `"confidence":0.97`) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestHealthDoesNotLeakConfig(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewRouter(RouterOptions{Classify: http.NotFoundHandler()}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}
