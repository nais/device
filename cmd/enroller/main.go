package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/internal/auth"
	"github.com/nais/device/internal/program"
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
		log.WithError(err).Error("main")
		os.Exit(1)
	} else {
		log.Error("main finished")
	}
}

func run(cfg Config, log *logrus.Entry) error {
	ctx, cancel := program.MainContext(5 * time.Second)
	defer cancel()

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

	tokenValidator, err := makeTokenValidator(ctx, cfg, log.WithField("component", "tokenValidator"))
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

	log.WithField("address", ":"+port).Info("starting server")
	go logErr(log, cancel, func() error { return worker.Run(ctx) })
	go logErr(log, cancel, server.ListenAndServe)

	<-ctx.Done()

	// Give 5s more to process existing requests
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("http server shutdown")
	}

	return nil
}

func logErr(log *logrus.Entry, cancel context.CancelFunc, fn func() error) {
	if err := fn(); err != nil {
		cancel()
		log.WithError(err).Error("closing")
	}
}

func makeWorker(cfg Config, ctx context.Context, log *logrus.Entry) (pubsubenroll.Worker, error) {
	if cfg.AzureEnabled || cfg.GoogleEnabled {
		return pubsubenroll.NewWorker(ctx, log)
	} else {
		log.Warn("AUTH DISABLED, this should NOT run in production")
		return pubsubenroll.NewNoopWorker(context.Background(), log), nil
	}
}

func makeTokenValidator(ctx context.Context, cfg Config, log *logrus.Entry) (auth.TokenValidator, error) {
	if cfg.AzureEnabled {
		err := cfg.Azure.SetupJwkCache(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetch Azure certs: %w", err)
		}

		return cfg.Azure.TokenValidatorMiddleware(), nil
	} else if cfg.GoogleEnabled {
		err := cfg.Google.SetupJwkSetAutoRefresh(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetch Google certs: %w", err)
		}

		return cfg.Google.TokenValidatorMiddleware(), nil
	} else {
		log.Warn("AUTH DISABLED, this should NOT run in production")
		return auth.MockTokenValidator(), nil
	}
}
