package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/internal/auth"
	"github.com/nais/device/internal/pubsubenroll"
	"github.com/sirupsen/logrus"
)

type Config struct {
	AzureEnabled  bool
	Azure         *auth.Azure
	GoogleEnabled bool
	Google        *auth.Google
	Production    bool
}

func main() {
	cfg := Config{
		Azure: &auth.Azure{
			ClientID: "6e45010d-2637-4a40-b91d-d4cbb451fb57",
			Tenant:   "62366534-1ec3-4962-8869-9b5535279d0b",
		},
		Google: &auth.Google{
			ClientID: "955023559628-g51n36t4icbd6lq7ils4r0ol9oo8kpk0.apps.googleusercontent.com",
		},
	}

	log := logrus.New()
	log.Formatter = &logrus.JSONFormatter{}

	if err := run(cfg, log.WithField("component", "main")); err != nil {
		log.Errorf("main: %v", err)
		os.Exit(1)
	} else {
		log.Errorf("main finished")
	}
}

func run(cfg Config, log *logrus.Entry) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	if err := envconfig.Process("ENROLLER", &cfg); err != nil {
		return fmt.Errorf("process envconfig: %w", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if cfg.AzureEnabled && cfg.GoogleEnabled {
		return fmt.Errorf("both Google and Azure auth enabled - pick one")
	}

	worker, err := makeWorker(cfg, ctx, log.WithField("component", "worker"))
	if err != nil {
		return fmt.Errorf("error setting up worker: %v", err)
	}

	tokenValidator, err := makeTokenValidator(cfg, log.WithField("component", "tokenValidator"))
	if err != nil {
		return fmt.Errorf("error setting up token validator: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/enroll", pubsubenroll.NewHandler(worker, log.WithField("component", "enroller")))

	server := http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 3 * time.Second,
		IdleTimeout:       10 * time.Minute,
		Handler:           tokenValidator(mux),
	}

	log.WithField("addr", ":"+port).Info("starting server")
	ctx, cancel := context.WithCancel(ctx)
	go logErr(log, cancel, func() error { return worker.Run(ctx) })
	go logErr(log, cancel, server.ListenAndServe)

	<-ctx.Done()

	// Reset os.Interrupt default behavior, similar to signal.Reset
	stop()
	log.Info("shutting down gracefully, press Ctrl+C again to force")

	// Give 5s more to process existing requests
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(timeoutCtx); err != nil {
		log.Error(err)
	}

	return nil
}

func logErr(log *logrus.Entry, cancel context.CancelFunc, fn func() error) {
	if err := fn(); err != nil {
		cancel()
		log.WithError(err).Error("error")
	}
}

func makeWorker(cfg Config, ctx context.Context, log *logrus.Entry) (pubsubenroll.Worker, error) {
	if cfg.AzureEnabled || cfg.GoogleEnabled {
		return pubsubenroll.NewWorker(ctx, log)
	} else {
		log.Warnf("AUTH DISABLED, this should NOT run in production")
		return pubsubenroll.NewNoopWorker(context.Background(), log), nil
	}
}

func makeTokenValidator(cfg Config, log *logrus.Entry) (auth.TokenValidator, error) {
	if cfg.AzureEnabled {
		err := cfg.Azure.SetupJwkSetAutoRefresh()
		if err != nil {
			return nil, fmt.Errorf("fetch Azure certs: %w", err)
		}

		return cfg.Azure.TokenValidatorMiddleware(), nil
	} else if cfg.GoogleEnabled {
		err := cfg.Google.SetupJwkSetAutoRefresh()
		if err != nil {
			return nil, fmt.Errorf("fetch Google certs: %w", err)
		}

		return cfg.Google.TokenValidatorMiddleware(), nil
	} else {
		log.Warnf("AUTH DISABLED, this should NOT run in production")
		return auth.MockTokenValidator(), nil
	}
}
