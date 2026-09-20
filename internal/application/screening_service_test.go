package application

import (
	"context"
	"errors"
	"testing"

	"github.com/SuchintK/jev-call-screener/internal/adapters/mockclassifier"
	"github.com/SuchintK/jev-call-screener/internal/domain"
)

func TestScreeningRouting(t *testing.T) {
	policy := RoutingPolicy{PromotionalRejectThreshold: 0.90, WantedForwardThreshold: 0.75, MaxClarificationTurns: 1, ForwardOnError: true}
	tests := []struct {
		name           string
		classification domain.Classification
		turns          int
		want           domain.Action
	}{
		{"high-confidence promotional rejects", domain.Classification{Category: domain.CategoryPromotional, Confidence: .97}, 0, domain.ActionReject},
		{"high-confidence wanted forwards", domain.Classification{Category: domain.CategoryWanted, Confidence: .80}, 0, domain.ActionForward},
		{"low confidence clarifies", domain.Classification{Category: domain.CategoryPromotional, Confidence: .60}, 0, domain.ActionClarify},
		{"uncertain after clarification forwards", domain.Classification{Category: domain.CategoryUnclear, Confidence: .70}, 1, domain.ActionForward},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			classifier := mockclassifier.New(mockclassifier.WithFallback(test.classification))
			decision := NewScreeningService(classifier, policy).Screen(context.Background(), "caller speech", test.turns)
			if decision.Action != test.want {
				t.Fatalf("got %q, want %q", decision.Action, test.want)
			}
		})
	}
}

func TestClassifierErrorFailsOpen(t *testing.T) {
	classifier := mockclassifier.New(mockclassifier.WithError(errors.New("unavailable")))
	policy := RoutingPolicy{MaxClarificationTurns: 1, ForwardOnError: true}
	decision := NewScreeningService(classifier, policy).Screen(context.Background(), "important call", 0)
	if decision.Action != domain.ActionForward || decision.Reason != "classifier_error_fail_open" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}
