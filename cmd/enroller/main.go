package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/pkg/auth"
	"github.com/nais/device/pkg/pubsubenroll"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Azure  *auth.Azure
	Google *auth.Google
}

func main() {
	cfg := Config{}

	if err := envconfig.Process("ENROLLER", &cfg); err != nil {
		logrus.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	log := logrus.New()
	log.Formatter = &logrus.JSONFormatter{}

	worker, err := pubsubenroll.NewWorker(context.Background(), log.WithField("component", "worker"))
	if err != nil {
		log.WithError(err).Fatal("new worker")
		return
	}

	var tokenValidator auth.TokenValidator
	if cfg.Azure != nil && cfg.Google != nil {
		log.Fatal("Both Google and Azure auth enabled - pick one")
	}

	if cfg.Azure != nil {
		err := cfg.Azure.FetchCertificates()
		if err != nil {
			log.Fatalf("fetch Azure certs: %s", err)
		}

		tokenValidator = cfg.Azure.TokenValidatorMiddleware()
	} else if cfg.Google != nil {
		err := cfg.Google.SetupJwkAutoRefresh()
		if err != nil {
			log.Fatalf("fetch Google certs: %s", err)
		}

		tokenValidator = cfg.Google.TokenValidatorMiddleware()
	} else {
		log.Warnf("AUTH DISABLED, this should NOT run in production")
		tokenValidator = auth.MockTokenValidator()
	}

	h := pubsubenroll.NewHandler(worker, log.WithField("component", "enroller"))

	mux := http.NewServeMux()
	mux.Handle("/enroll", h)

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

	// Gievn 5s more to process existing requests
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(timeoutCtx); err != nil {
		log.Error(err)
	}
}

func logErr(log *logrus.Logger, cancel context.CancelFunc, fn func() error) {
	if err := fn(); err != nil {
		cancel()
		log.WithError(err).Error("error")
	}
}
