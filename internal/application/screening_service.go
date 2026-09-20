package application

import (
	"context"
	"strings"

	"github.com/SuchintK/jev-call-screener/internal/domain"
	"github.com/SuchintK/jev-call-screener/internal/ports"
)

// ScreeningService coordinates classification and routing without telephony dependencies.
type ScreeningService struct {
	classifier ports.Classifier
	policy     RoutingPolicy
}

func NewScreeningService(classifier ports.Classifier, policy RoutingPolicy) *ScreeningService {
	return &ScreeningService{classifier: classifier, policy: policy}
}

// Screen classifies a transcript and applies routing policy. Empty speech is treated
// as uncertain rather than sent to the remote classifier.
func (s *ScreeningService) Screen(ctx context.Context, transcript string, clarificationTurns int) domain.Decision {
	transcript = strings.TrimSpace(transcript)
	if transcript == "" {
		return s.policy.Decide(domain.Classification{Category: domain.CategoryUnclear}, clarificationTurns)
	}
	classification, err := s.classifier.Classify(ctx, transcript)
	if err != nil {
		return s.policy.OnClassifierError(clarificationTurns)
	}
	return s.policy.Decide(classification, clarificationTurns)
}
