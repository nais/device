package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/internal/logger"
	"github.com/nais/device/internal/program"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

type Config struct {
	ClientSecret string
	ClientID     string
}

type ExchangeRequest struct {
	CodeVerifier string `json:"code_verifier"`
	AccessCode   string `json:"access_code"`
	RedirectURI  string `json:"redirect_uri"`
}

type ExchangeResponse struct {
	Token   *oauth2.Token `json:"token"`
	IDToken string        `json:"id_token"`
}

func main() {
	ctx, cancel := program.MainContext(time.Second)
	defer cancel()

	cfg := &Config{}
	log := logger.Setup(logrus.InfoLevel.String()).WithField("component", "main")
	err := envconfig.Process("AUTH_SERVER", cfg)
	if err != nil {
		log.WithError(err).Fatal("process envconfig")
	}

	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.ClientSecret = strings.TrimSpace(cfg.ClientSecret)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	bind := ":" + port

	baseOAuth2Config := &oauth2.Config{
		ClientSecret: cfg.ClientSecret,
		ClientID:     cfg.ClientID,
		Scopes:       []string{".default"},
		Endpoint:     endpoints.Google,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/exchange", exchange(log, baseOAuth2Config))
	log.WithField("address", bind).Info("listening")

	server := &http.Server{
		Addr:    bind,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.WithError(err).Warn("server closed for unknown reason")
		}
	}()

	<-ctx.Done()

	// Give 5s more to process existing requests
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("http server shutdown")
	}
}

func exchange(log *logrus.Entry, oauth2config *oauth2.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var exchangeData ExchangeRequest
		err := json.NewDecoder(r.Body).Decode(&exchangeData)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.WithError(err).Warn("decode exchange data")
			return
		}

		codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", exchangeData.CodeVerifier)
		redirectURIParam := oauth2.SetAuthURLParam("redirect_uri", exchangeData.RedirectURI)
		token, err := oauth2config.Exchange(r.Context(), exchangeData.AccessCode, codeVerifierParam, redirectURIParam)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.WithError(err).Warn("exchange code for token")
			return
		}

		err = json.NewEncoder(w).Encode(ExchangeResponse{
			Token:   token,
			IDToken: token.Extra("id_token").(string),
		})
		if err != nil {
			log.WithError(err).Warn("encode response")
			return
		}

		log.Info("successfully returned token")
	}
}
