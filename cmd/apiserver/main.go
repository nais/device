package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/nais/device/apiserver/api"
	"github.com/nais/device/apiserver/config"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/apiserver/slack"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = config.DefaultConfig()
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	flag.StringVar(&cfg.DbConnURI, "db-connection-uri", os.Getenv("DB_CONNECTION_URI"), "database connection URI (DSN)")
	flag.StringVar(&cfg.SlackToken, "slack-token", os.Getenv("SLACK_TOKEN"), "Slack token")
	flag.StringVar(&cfg.BindAddress, "bind-address", "10.255.240.1:80", "Bind address")

	flag.Parse()
}

func main() {
	db, err := database.New(cfg.DbConnURI)

	if err != nil {
		panic(fmt.Sprintf("instantiating database: %s", err))
	}


	slack:= slack.New(cfg.SlackToken, db)
	slack.Run()

	router := api.New(api.Config{DB: db})

	fmt.Println("running @", cfg.BindAddress)
	fmt.Println(http.ListenAndServe(cfg.BindAddress, router))
}
