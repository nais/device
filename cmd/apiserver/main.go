package main

import (
	"fmt"
	"net/http"

	"github.com/nais/device/apiserver/api"
	"github.com/nais/device/apiserver/config"
	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = config.DefaultConfig()
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	flag.StringVar(&cfg.DbConnURI, "db-connection-uri", cfg.DbConnURI, "database connection URI (DSN)")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "database connection URI (DSN)")

	flag.Parse()
}

func main() {
	db, err := database.New(cfg.DbConnURI)

	if err != nil {
		panic(fmt.Sprintf("instantiating database: %s", err))
	}

	router := api.New(api.Config{DB: db})

	fmt.Println("running @", cfg.BindAddress)
	fmt.Println(http.ListenAndServe(cfg.BindAddress, router))
}
