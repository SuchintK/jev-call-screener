package config

import (
	"strings"
	"testing"
)

func baseline(t *testing.T) {
	t.Helper()
	for _, name := range []string{"TYPESAFE_API_KEY", "FORWARD_TO_NUMBER", "TWILIO_AUTH_TOKEN", "PUBLIC_BASE_URL", "SCREENING_CONFIG_PATH"} {
		t.Setenv(name, "")
	}
	t.Setenv("TELEPHONY_PROVIDER", "none")
	t.Setenv("PROMOTIONAL_REJECT_THRESHOLD", "0.90")
	t.Setenv("WANTED_FORWARD_THRESHOLD", "0.75")
	t.Setenv("JEV_TIMEOUT_MS", "3000")
	t.Setenv("MAX_CLARIFICATION_TURNS", "1")
	t.Setenv("FORWARD_ON_ERROR", "true")
	t.Setenv("TWILIO_VALIDATE_SIGNATURE", "true")
	t.Setenv("LOG_TRANSCRIPTS", "false")
}

func TestMissingTypeSafeKey(t *testing.T) {
	baseline(t)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TYPESAFE_API_KEY is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestInvalidThreshold(t *testing.T) {
	baseline(t)
	t.Setenv("TYPESAFE_API_KEY", "test")
	t.Setenv("PROMOTIONAL_REJECT_THRESHOLD", "1.2")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "PROMOTIONAL_REJECT_THRESHOLD") {
		t.Fatalf("error = %v", err)
	}
}

func TestMissingForwardingNumberForTwilio(t *testing.T) {
	baseline(t)
	t.Setenv("TYPESAFE_API_KEY", "test")
	t.Setenv("TELEPHONY_PROVIDER", "twilio")
	t.Setenv("TWILIO_VALIDATE_SIGNATURE", "false")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "FORWARD_TO_NUMBER") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidNonTelephonyConfig(t *testing.T) {
	baseline(t)
	t.Setenv("TYPESAFE_API_KEY", "test")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JEVModel != DefaultJEVModel || cfg.MaxClarificationTurns != 1 {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
}
