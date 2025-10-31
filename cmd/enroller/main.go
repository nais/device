package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/internal/enroll"
	"github.com/nais/device/internal/program"
	"github.com/nais/device/internal/token"
	"github.com/nais/device/internal/token/azure"
	"github.com/nais/device/internal/token/google"
	"github.com/sirupsen/logrus"
)

type Config struct {
	AzureEnabled    bool
	Azure           token.Config
	GoogleEnabled   bool
	Google          token.Config
	LocalListenAddr string
}

func main() {
	cfg := Config{
		Azure:           azure.APIServerConfig,
		Google:          google.APIServerConfig,
		LocalListenAddr: "",
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
	mux.Handle("/enroll", enroll.NewHandler(worker, log.WithField("component", "enroller")))

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

func makeWorker(cfg Config, ctx context.Context, log *logrus.Entry) (enroll.Worker, error) {
	if cfg.LocalListenAddr != "" {
		log.Warn("LOCAL MODE, this should NOT run in production")
		return enroll.NewLocal(ctx, cfg.LocalListenAddr, log)
	}

	if cfg.AzureEnabled || cfg.GoogleEnabled {
		return enroll.NewPubSub(ctx, log)
	}

	return enroll.NewNoopWorker(context.Background(), log), nil
}

func makeTokenValidator(ctx context.Context, cfg Config, log *logrus.Entry) (token.Validator, error) {
	if cfg.AzureEnabled {
		return token.Middleware(azure.New(ctx, cfg.Azure)), nil
	} else if cfg.GoogleEnabled {
		return token.Middleware(google.New(ctx, cfg.Google)), nil
	} else {
		log.Warn("AUTH DISABLED, this should NOT run in production")
		return token.MockValidator(), nil
	}
}
