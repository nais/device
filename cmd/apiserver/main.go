package main

import (
	"fmt"
	"net/http"
	"os"

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
	flag.StringVar(&cfg.DbConnURI, "db-connection-uri", os.Getenv("DB_CONNECTION_URI"), "database connection URI (DSN)")

	flag.Parse()
}

func main() {
	db, err := database.New(cfg.DbConnURI)

	if err != nil {
		panic(fmt.Sprintf("instantiating database: %s", err))
	}

	router := api.New(api.Config{DB: db})

	bindAddr := fmt.Sprintf("%s:%d", "127.0.0.1", 8080)
	fmt.Println("running @", bindAddr)
	fmt.Println(http.ListenAndServe(bindAddr, router))
}
