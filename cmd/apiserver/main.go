package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	_ "github.com/lib/pq"
)

var (
	dbUser     string
	dbPassword string
	dbHost     string
	dbName     string
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	flag.StringVar(&dbName, "db-name", getEnv("DB_NAME", "postgres"), "database name")
	flag.StringVar(&dbUser, "db-user", getEnv("DB_USER", "postgres"), "database username")
	flag.StringVar(&dbPassword, "db-password", os.Getenv("DB_PASSWORD"), "database password")
	flag.StringVar(&dbHost, "db-hostname", getEnv("DB_HOST", "localhost"), "database hostname")
	flag.Parse()
}

type Client struct {
	PSK string `json:"psk"`
	Peer
}

type Peer struct {
	PublicKey string `json:"public_key"`
	IP        string `json:"ip"`
}

type GatewayResponse struct {
	Clients []Client `json:"clients"`
}

func main() {
	http.HandleFunc("/gateways/gw0", gatewayConfigHandler())

	bindAddr := fmt.Sprintf("%s:%d", "127.0.0.1", 6969)
	fmt.Println("running @", bindAddr)
	fmt.Println((&http.Server{Addr: bindAddr}).ListenAndServe())
}

func gatewayConfigHandler() func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		postgresConnection := fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=disable",
			dbUser,
			dbPassword,
			dbName,
			dbHost)

		db, err := sql.Open("postgres", postgresConnection)
		if err != nil {
			panic(fmt.Sprintf("failed to connect to database, error was: %s", err))
		}

		rows, err := db.Query(`
            SELECT public_key, ip, psk from peer
            JOIN client c on peer.id = c.peer_id
            JOIN ip i on peer.id = i.peer_id
		`)

		if err != nil {
			panic(err)
		}

		var resp GatewayResponse

		for rows.Next() {
			var client Client

			err := rows.Scan(&client.PublicKey, &client.IP, &client.PSK)

			if err != nil {
				panic(err)
			}

			resp.Clients = append(resp.Clients, client)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}
