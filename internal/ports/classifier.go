package ports

import (
	"context"

	"github.com/SuchintK/jev-call-screener/internal/domain"
)

// Classifier classifies untrusted caller speech without deciding call routing.
type Classifier interface {
	Classify(ctx context.Context, transcript string) (domain.Classification, error)
}
