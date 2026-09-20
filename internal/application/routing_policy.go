package application

import "github.com/SuchintK/jev-call-screener/internal/domain"

// RoutingPolicy contains deliberately simple and auditable routing rules.
type RoutingPolicy struct {
	PromotionalRejectThreshold float64
	WantedForwardThreshold     float64
	MaxClarificationTurns      int
	ForwardOnError             bool
}

func (p RoutingPolicy) Decide(c domain.Classification, clarificationTurns int) domain.Decision {
	switch {
	case c.Category == domain.CategoryPromotional && c.Confidence >= p.PromotionalRejectThreshold:
		return domain.Decision{Action: domain.ActionReject, Classification: c, Reason: "high_confidence_promotional"}
	case c.Category == domain.CategoryWanted && c.Confidence >= p.WantedForwardThreshold:
		return domain.Decision{Action: domain.ActionForward, Classification: c, Reason: "high_confidence_wanted"}
	case clarificationTurns < p.MaxClarificationTurns:
		return domain.Decision{Action: domain.ActionClarify, Classification: c, Reason: "more_information_needed"}
	default:
		return domain.Decision{Action: domain.ActionForward, Classification: c, Reason: "uncertainty_fail_open"}
	}
}

func (p RoutingPolicy) OnClassifierError(clarificationTurns int) domain.Decision {
	c := domain.Classification{Category: domain.CategoryUnclear}
	if p.ForwardOnError {
		return domain.Decision{Action: domain.ActionForward, Classification: c, Reason: "classifier_error_fail_open"}
	}
	if clarificationTurns < p.MaxClarificationTurns {
		return domain.Decision{Action: domain.ActionClarify, Classification: c, Reason: "classifier_error_clarify"}
	}
	return domain.Decision{Action: domain.ActionReject, Classification: c, Reason: "classifier_error_configured_closed"}
}
