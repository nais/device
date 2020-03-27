package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	_ "github.com/lib/pq"
)

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
	http.HandleFunc("/gateways/gw0", func(w http.ResponseWriter, _ *http.Request) {
		postgresConnection := fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=disable",
			"postgres",
			"asdf",
			"postgres",
			"localhost")

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
	})

	bindAddr := fmt.Sprintf("%s:%d", "127.0.0.1", 6969)
	fmt.Println("running @", bindAddr)
	fmt.Println((&http.Server{Addr: bindAddr}).ListenAndServe())
}
