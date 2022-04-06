package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
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
	cfg := &Config{}
	err := envconfig.Process("AUTH_SERVER", cfg)
	if err != nil {
		log.Fatalf("process envconfig: %s", err)
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
	mux.HandleFunc("/exchange", exchange(baseOAuth2Config))
	log.WithField("bind", bind).Info("listening")
	err = http.ListenAndServe(bind, mux)
	if err != http.ErrServerClosed {
		log.WithError(err).Warn("server closed for unknown reason")
	}

	log.Info("finished")
}

func exchange(oauth2config *oauth2.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var exchangeData ExchangeRequest
		err := json.NewDecoder(r.Body).Decode(&exchangeData)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.WithError(err).Warnf("decode exchange data")
			return
		}

		codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", exchangeData.CodeVerifier)
		redirectURIParam := oauth2.SetAuthURLParam("redirect_uri", exchangeData.RedirectURI)
		token, err := oauth2config.Exchange(r.Context(), exchangeData.AccessCode, codeVerifierParam, redirectURIParam)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.WithError(err).Warnf("exchange code for token")
			return
		}

		err = json.NewEncoder(w).Encode(ExchangeResponse{
			Token:   token,
			IDToken: token.Extra("id_token").(string),
		})
		if err != nil {
			log.WithError(err).Warnf("encode response")
			return
		}

		log.Infof("successfully returned token")
	}
}
