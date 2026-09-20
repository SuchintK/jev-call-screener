package api

import "net/http"

type RouterOptions struct {
	Classify       http.Handler
	TwilioIncoming http.Handler
	TwilioGather   http.Handler
}

func NewRouter(options RouterOptions) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.Handle("POST /api/v1/classify", options.Classify)
	if options.TwilioIncoming != nil {
		mux.Handle("POST /webhooks/twilio/incoming", options.TwilioIncoming)
	}
	if options.TwilioGather != nil {
		mux.Handle("POST /webhooks/twilio/gather", options.TwilioGather)
	}
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
