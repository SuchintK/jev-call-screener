package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SuchintK/jev-call-screener/internal/adapters/jev"
	"github.com/SuchintK/jev-call-screener/internal/adapters/memory"
	"github.com/SuchintK/jev-call-screener/internal/adapters/twilio"
	"github.com/SuchintK/jev-call-screener/internal/api"
	"github.com/SuchintK/jev-call-screener/internal/application"
	"github.com/SuchintK/jev-call-screener/internal/config"
	"github.com/SuchintK/jev-call-screener/internal/domain"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	descriptions := make(map[domain.Category]string, len(cfg.Screening.Categories))
	for name, rule := range cfg.Screening.Categories {
		descriptions[domain.Category(name)] = rule.Description
	}
	classifier := jev.NewClient(cfg.TypeSafeAPIKey, cfg.JEVModel, cfg.JEVTimeout, jev.WithCategoryDescriptions(descriptions))
	service := application.NewScreeningService(classifier, application.RoutingPolicy{
		PromotionalRejectThreshold: cfg.PromotionalRejectThreshold,
		WantedForwardThreshold:     cfg.WantedForwardThreshold,
		MaxClarificationTurns:      cfg.MaxClarificationTurns,
		ForwardOnError:             cfg.ForwardOnError,
	})

	options := api.RouterOptions{Classify: api.ClassifyHandler{Service: service}}
	if cfg.TelephonyProvider == "twilio" {
		sessions := memory.NewSessionStore()
		incoming := http.Handler(twilio.IncomingHandler{Sessions: sessions, Greeting: cfg.Screening.Messages.Greeting})
		gather := http.Handler(twilio.GatherHandler{
			Service:             service,
			Sessions:            sessions,
			ForwardToNumber:     cfg.ForwardToNumber,
			ClarificationPrompt: cfg.Screening.Messages.Clarification,
			RejectedMessage:     cfg.Screening.Messages.Rejected,
			LogTranscripts:      cfg.LogTranscripts,
			Logger:              logger,
		})
		if cfg.TwilioValidateSignature {
			validator := twilio.SignatureValidator{AuthToken: cfg.TwilioAuthToken, PublicBaseURL: cfg.PublicBaseURL}
			incoming = twilio.ValidateSignatures(validator, incoming)
			gather = twilio.ValidateSignatures(validator, gather)
		}
		options.TwilioIncoming = incoming
		options.TwilioGather = gather
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.NewRouter(options),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-shutdownContext.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("server shutdown failed", "error", err)
		}
	}()

	logger.Info("server listening", "address", server.Addr, "telephony_provider", cfg.TelephonyProvider, "jev_model", cfg.JEVModel)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
