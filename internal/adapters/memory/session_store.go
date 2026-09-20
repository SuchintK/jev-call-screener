package memory

import (
	"context"
	"sync"

	"github.com/SuchintK/jev-call-screener/internal/domain"
)

// SessionStore is a process-local session store suitable for a single V1 instance.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]domain.Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]domain.Session)}
}

func (s *SessionStore) Get(_ context.Context, id string) (domain.Session, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	if ok {
		session.Transcripts = append([]string(nil), session.Transcripts...)
	}
	return session, ok, nil
}

func (s *SessionStore) Save(_ context.Context, session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session.Transcripts = append([]string(nil), session.Transcripts...)
	s.sessions[session.ID] = session
	return nil
}

func (s *SessionStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}
