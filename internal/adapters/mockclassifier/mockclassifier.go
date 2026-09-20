package mockclassifier

import (
	"context"
	"strings"

	"github.com/SuchintK/jev-call-screener/internal/domain"
)

type response struct {
	needle string
	value  domain.Classification
	err    error
}

// Option configures deterministic substring matching.
type Option func(*Classifier)

// Classifier is a deterministic test adapter. First matching response wins.
type Classifier struct {
	responses []response
	fallback  domain.Classification
	err       error
}

func New(options ...Option) *Classifier {
	c := &Classifier{fallback: domain.Classification{Category: domain.CategoryUnclear, Confidence: 0.5, Model: "mock"}}
	for _, option := range options {
		option(c)
	}
	return c
}

func WithResponse(substring string, classification domain.Classification) Option {
	return func(c *Classifier) {
		c.responses = append(c.responses, response{needle: strings.ToLower(substring), value: classification})
	}
}

func WithError(err error) Option {
	return func(c *Classifier) { c.err = err }
}

func WithFallback(classification domain.Classification) Option {
	return func(c *Classifier) { c.fallback = classification }
}

func (c *Classifier) Classify(_ context.Context, transcript string) (domain.Classification, error) {
	if c.err != nil {
		return domain.Classification{}, c.err
	}
	lower := strings.ToLower(transcript)
	for _, candidate := range c.responses {
		if strings.Contains(lower, candidate.needle) {
			return candidate.value, candidate.err
		}
	}
	return c.fallback, nil
}
