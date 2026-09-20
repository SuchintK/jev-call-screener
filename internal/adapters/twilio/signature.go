package twilio

import (
	"crypto/hmac"
	"crypto/sha1" // Twilio's webhook signature scheme specifies HMAC-SHA1.
	"encoding/base64"
	"net/http"
	"sort"
	"strings"
)

// SignatureValidator validates Twilio's X-Twilio-Signature without an SDK dependency.
type SignatureValidator struct {
	AuthToken     string
	PublicBaseURL string
}

func (v SignatureValidator) Valid(r *http.Request) bool {
	provided, err := base64.StdEncoding.DecodeString(r.Header.Get("X-Twilio-Signature"))
	if err != nil || len(provided) == 0 || r.ParseForm() != nil {
		return false
	}
	url := strings.TrimRight(v.PublicBaseURL, "/") + r.URL.RequestURI()
	keys := make([]string, 0, len(r.PostForm))
	for key := range r.PostForm {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var signed strings.Builder
	signed.WriteString(url)
	for _, key := range keys {
		values := append([]string(nil), r.PostForm[key]...)
		sort.Strings(values)
		for _, value := range values {
			signed.WriteString(key)
			signed.WriteString(value)
		}
	}
	mac := hmac.New(sha1.New, []byte(v.AuthToken))
	_, _ = mac.Write([]byte(signed.String()))
	return hmac.Equal(provided, mac.Sum(nil))
}

func ValidateSignatures(validator SignatureValidator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		if !validator.Valid(r) {
			http.Error(w, "invalid webhook signature", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
