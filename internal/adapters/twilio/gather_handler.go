package twilio

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/SuchintK/jev-call-screener/internal/application"
	"github.com/SuchintK/jev-call-screener/internal/domain"
	"github.com/SuchintK/jev-call-screener/internal/ports"
)

type GatherHandler struct {
	Service             *application.ScreeningService
	Sessions            ports.SessionStore
	ForwardToNumber     string
	ClarificationPrompt string
	RejectedMessage     string
	LogTranscripts      bool
	Logger              *slog.Logger
}

func (h GatherHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("CallSid"))
	if id == "" {
		// Without a provider ID clarification cannot be correlated, so fail open.
		writeXML(w, forwardXML(h.ForwardToNumber))
		return
	}

	session, found, err := h.Sessions.Get(r.Context(), id)
	if err != nil {
		h.logError("session lookup failed", err)
		writeXML(w, forwardXML(h.ForwardToNumber))
		return
	}
	if !found {
		session = domain.Session{ID: id, CreatedAt: time.Now().UTC()}
	}
	speech := strings.TrimSpace(r.FormValue("SpeechResult"))
	if speech != "" {
		session.Transcripts = append(session.Transcripts, speech)
	}
	transcript := combinedTranscript(session.Transcripts)
	if h.LogTranscripts && h.Logger != nil {
		h.Logger.Info("screening caller transcript", "transcript", transcript)
	}
	decision := h.Service.Screen(r.Context(), transcript, session.TurnCount)

	switch decision.Action {
	case domain.ActionReject:
		_ = h.Sessions.Delete(r.Context(), id)
		writeXML(w, rejectXML(h.RejectedMessage))
	case domain.ActionClarify:
		session.TurnCount++
		if err := h.Sessions.Save(r.Context(), session); err != nil {
			h.logError("session save failed", err)
			writeXML(w, forwardXML(h.ForwardToNumber))
			return
		}
		writeXML(w, gatherXML(h.ClarificationPrompt))
	default:
		_ = h.Sessions.Delete(r.Context(), id)
		writeXML(w, forwardXML(h.ForwardToNumber))
	}
}

func combinedTranscript(transcripts []string) string {
	if len(transcripts) == 0 {
		return ""
	}
	if len(transcripts) == 1 {
		return transcripts[0]
	}
	return fmt.Sprintf("Caller initial response:\n%s\n\nCaller clarification:\n%s", transcripts[0], strings.Join(transcripts[1:], "\n"))
}

func (h GatherHandler) logError(message string, err error) {
	if h.Logger != nil {
		h.Logger.ErrorContext(context.Background(), message, "error", err)
	}
}
