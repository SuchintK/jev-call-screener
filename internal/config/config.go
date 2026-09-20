package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultJEVModel = "jev-1.13.0"

type CategoryRule struct {
	Description string `yaml:"description"`
}

type Messages struct {
	Greeting      string `yaml:"greeting"`
	Clarification string `yaml:"clarification"`
	Rejected      string `yaml:"rejected"`
}

type Screening struct {
	Categories map[string]CategoryRule `yaml:"categories"`
	Messages   Messages                `yaml:"messages"`
}

type Config struct {
	Port                       string
	PublicBaseURL              string
	TypeSafeAPIKey             string
	JEVModel                   string
	JEVTimeout                 time.Duration
	PromotionalRejectThreshold float64
	WantedForwardThreshold     float64
	MaxClarificationTurns      int
	ForwardOnError             bool
	TelephonyProvider          string
	TwilioAccountSID           string
	TwilioAuthToken            string
	TwilioPhoneNumber          string
	ForwardToNumber            string
	TwilioValidateSignature    bool
	LogTranscripts             bool
	Screening                  Screening
}

func defaultScreening() Screening {
	return Screening{
		Categories: map[string]CategoryRule{
			"promotional": {Description: "Unsolicited sales, marketing, product offers, loans, credit cards, insurance, subscriptions, upgrades, or other attempts to sell something."},
			"wanted":      {Description: "Deliveries requiring action, recruiters, interviews, appointments, personal communication, existing account issues, and support for an existing service."},
			"unclear":     {Description: "Not enough information is available to confidently determine the purpose of the call."},
		},
		Messages: Messages{
			Greeting:      "Hi. This call is being screened. Please briefly tell me what you're calling about.",
			Clarification: "Could you briefly provide a little more detail about the reason for your call?",
			Rejected:      "Thanks. They're not accepting promotional calls right now.",
		},
	}
}

// Load reads non-secret screening preferences from YAML and all credentials from the environment.
func Load() (Config, error) {
	cfg := Config{
		Port:                       env("PORT", "8080"),
		PublicBaseURL:              strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/"),
		TypeSafeAPIKey:             os.Getenv("TYPESAFE_API_KEY"),
		JEVModel:                   env("JEV_MODEL", DefaultJEVModel),
		PromotionalRejectThreshold: 0.90,
		WantedForwardThreshold:     0.75,
		MaxClarificationTurns:      1,
		ForwardOnError:             true,
		TelephonyProvider:          strings.ToLower(env("TELEPHONY_PROVIDER", "twilio")),
		TwilioAccountSID:           os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:            os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioPhoneNumber:          os.Getenv("TWILIO_PHONE_NUMBER"),
		ForwardToNumber:            os.Getenv("FORWARD_TO_NUMBER"),
		TwilioValidateSignature:    true,
		LogTranscripts:             false,
		Screening:                  defaultScreening(),
	}

	var errs []error
	if value, err := intEnv("JEV_TIMEOUT_MS", 3000); err != nil || value <= 0 {
		errs = append(errs, valueError("JEV_TIMEOUT_MS", err, "must be a positive integer"))
	} else {
		cfg.JEVTimeout = time.Duration(value) * time.Millisecond
	}
	cfg.PromotionalRejectThreshold, errs = floatEnv("PROMOTIONAL_REJECT_THRESHOLD", cfg.PromotionalRejectThreshold, errs)
	cfg.WantedForwardThreshold, errs = floatEnv("WANTED_FORWARD_THRESHOLD", cfg.WantedForwardThreshold, errs)
	cfg.MaxClarificationTurns, errs = integerEnv("MAX_CLARIFICATION_TURNS", cfg.MaxClarificationTurns, errs)
	cfg.ForwardOnError, errs = booleanEnv("FORWARD_ON_ERROR", cfg.ForwardOnError, errs)
	cfg.TwilioValidateSignature, errs = booleanEnv("TWILIO_VALIDATE_SIGNATURE", cfg.TwilioValidateSignature, errs)
	cfg.LogTranscripts, errs = booleanEnv("LOG_TRANSCRIPTS", cfg.LogTranscripts, errs)

	if path := os.Getenv("SCREENING_CONFIG_PATH"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("SCREENING_CONFIG_PATH: %w", err))
		} else {
			decoder := yaml.NewDecoder(strings.NewReader(string(data)))
			decoder.KnownFields(true)
			if err := decoder.Decode(&cfg.Screening); err != nil {
				errs = append(errs, fmt.Errorf("SCREENING_CONFIG_PATH: invalid YAML: %w", err))
			}
		}
	}

	errs = append(errs, cfg.validate()...)
	return cfg, errors.Join(errs...)
}

func (c Config) validate() []error {
	var errs []error
	port, portErr := strconv.Atoi(c.Port)
	if portErr != nil || port < 1 || port > 65535 {
		errs = append(errs, errors.New("PORT must be an integer between 1 and 65535"))
	}
	if strings.TrimSpace(c.TypeSafeAPIKey) == "" {
		errs = append(errs, errors.New("TYPESAFE_API_KEY is required"))
	}
	if c.JEVModel == "" {
		errs = append(errs, errors.New("JEV_MODEL must not be empty"))
	}
	if c.PromotionalRejectThreshold < 0 || c.PromotionalRejectThreshold > 1 {
		errs = append(errs, errors.New("PROMOTIONAL_REJECT_THRESHOLD must be between 0 and 1"))
	}
	if c.WantedForwardThreshold < 0 || c.WantedForwardThreshold > 1 {
		errs = append(errs, errors.New("WANTED_FORWARD_THRESHOLD must be between 0 and 1"))
	}
	if c.MaxClarificationTurns < 0 {
		errs = append(errs, errors.New("MAX_CLARIFICATION_TURNS must be zero or greater"))
	}
	if c.TelephonyProvider != "twilio" && c.TelephonyProvider != "none" {
		errs = append(errs, errors.New("TELEPHONY_PROVIDER must be twilio or none"))
	}
	if c.TelephonyProvider == "twilio" {
		if strings.TrimSpace(c.ForwardToNumber) == "" {
			errs = append(errs, errors.New("FORWARD_TO_NUMBER is required when TELEPHONY_PROVIDER=twilio"))
		}
		if c.TwilioValidateSignature && strings.TrimSpace(c.TwilioAuthToken) == "" {
			errs = append(errs, errors.New("TWILIO_AUTH_TOKEN is required when TWILIO_VALIDATE_SIGNATURE=true"))
		}
		if c.TwilioValidateSignature && strings.TrimSpace(c.PublicBaseURL) == "" {
			errs = append(errs, errors.New("PUBLIC_BASE_URL is required when TWILIO_VALIDATE_SIGNATURE=true"))
		} else if c.TwilioValidateSignature {
			parsed, err := url.Parse(c.PublicBaseURL)
			if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
				errs = append(errs, errors.New("PUBLIC_BASE_URL must be an absolute HTTPS URL"))
			}
		}
	}
	for _, name := range []string{"promotional", "wanted", "unclear"} {
		if strings.TrimSpace(c.Screening.Categories[name].Description) == "" {
			errs = append(errs, fmt.Errorf("screening category %q requires a description", name))
		}
	}
	if strings.TrimSpace(c.Screening.Messages.Greeting) == "" || strings.TrimSpace(c.Screening.Messages.Clarification) == "" || strings.TrimSpace(c.Screening.Messages.Rejected) == "" {
		errs = append(errs, errors.New("screening greeting, clarification, and rejected messages must not be empty"))
	}
	return errs
}

func env(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func intEnv(name string, fallback int) (int, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func floatEnv(name string, fallback float64, errs []error) (float64, []error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, errs
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback, append(errs, fmt.Errorf("%s must be a number", name))
	}
	return parsed, errs
}

func integerEnv(name string, fallback int, errs []error) (int, []error) {
	value, err := intEnv(name, fallback)
	if err != nil {
		return fallback, append(errs, fmt.Errorf("%s must be an integer", name))
	}
	return value, errs
}

func booleanEnv(name string, fallback bool, errs []error) (bool, []error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, errs
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback, append(errs, fmt.Errorf("%s must be true or false", name))
	}
	return parsed, errs
}

func valueError(name string, parseErr error, message string) error {
	if parseErr != nil {
		return fmt.Errorf("%s %s: %v", name, message, parseErr)
	}
	return fmt.Errorf("%s %s", name, message)
}
