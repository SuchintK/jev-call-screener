package twilio

import (
	"net/http"
	"time"

	"github.com/SuchintK/jev-call-screener/internal/domain"
	"github.com/SuchintK/jev-call-screener/internal/ports"
)

type IncomingHandler struct {
	Sessions ports.SessionStore
	Greeting string
}

func (h IncomingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	if id := r.FormValue("CallSid"); id != "" {
		_ = h.Sessions.Save(r.Context(), domain.Session{ID: id, CreatedAt: time.Now().UTC()})
	}
	writeXML(w, gatherXML(h.Greeting))
}

func writeXML(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", contentTypeXML)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
