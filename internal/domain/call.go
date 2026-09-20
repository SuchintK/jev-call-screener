package domain

import "time"

// Session stores the minimal provider-independent state needed for clarification.
type Session struct {
	ID          string
	TurnCount   int
	Transcripts []string
	CreatedAt   time.Time
}
