package ports

import (
	"context"

	"github.com/SuchintK/jev-call-screener/internal/domain"
)

// SessionStore persists short-lived call screening sessions.
type SessionStore interface {
	Get(ctx context.Context, id string) (domain.Session, bool, error)
	Save(ctx context.Context, session domain.Session) error
	Delete(ctx context.Context, id string) error
}
