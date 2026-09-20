package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/SuchintK/jev-call-screener/internal/application"
	"github.com/SuchintK/jev-call-screener/internal/domain"
)

type ClassifyHandler struct {
	Service *application.ScreeningService
}

type classifyRequest struct {
	Transcript string `json:"transcript"`
}

type classifyResponse struct {
	Classification domain.Classification `json:"classification"`
	Decision       responseDecision      `json:"decision"`
}

type responseDecision struct {
	Action domain.Action `json:"action"`
	Reason string        `json:"reason,omitempty"`
}

func (h ClassifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request classifyRequest
	if err := decoder.Decode(&request); err != nil || strings.TrimSpace(request.Transcript) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "a non-empty transcript is required"})
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must contain one JSON object"})
		return
	}
	decision := h.Service.Screen(r.Context(), request.Transcript, 0)
	writeJSON(w, http.StatusOK, classifyResponse{
		Classification: decision.Classification,
		Decision:       responseDecision{Action: decision.Action, Reason: decision.Reason},
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
