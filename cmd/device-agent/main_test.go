package main_test

import (
	"testing"

	main "github.com/nais/device/cmd/device-agent"
	"github.com/stretchr/testify/assert"
)

func TestParseBootstrapToken(t *testing.T) {
	/*
		{
		  "deviceIP": "10.1.1.1",
		  "publicKey": "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		  "tunnelEndpoint": "69.1.1.1:51820",
		  "apiServerIP": "10.1.1.2"
		}
	*/
	bootstrapToken := "ewogICJkZXZpY2VJUCI6ICIxMC4xLjEuMSIsCiAgInB1YmxpY0tleSI6ICJQUUttcmFQT1B5ZTVDSnExeDduanBsOHJSdTVSU3JJS3lIdlpYdEx2UzBFPSIsCiAgInR1bm5lbEVuZHBvaW50IjogIjY5LjEuMS4xOjUxODIwIiwKICAiYXBpU2VydmVySVAiOiAiMTAuMS4xLjIiCn0K"
	bootstrapConfig, err := main.ParseBootstrapToken(bootstrapToken)
	assert.NoError(t, err)
	assert.Equal(t, "10.1.1.1", bootstrapConfig.DeviceIP)
	assert.Equal(t, "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=", bootstrapConfig.PublicKey)
	assert.Equal(t, "69.1.1.1:51820", bootstrapConfig.Endpoint)
	assert.Equal(t, "10.1.1.2", bootstrapConfig.APIServerIP)
}

func TestGenerateWGConfig(t *testing.T) {
	bootstrapConfig := &main.BootstrapConfig{
		DeviceIP:    "10.1.1.1",
		PublicKey:   "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		Endpoint:    "69.1.1.1:51820",
		APIServerIP: "10.1.1.2",
	}
	privateKey := []byte("wFTAVe1stJPp0xQ+FE9so56uKh0jaHkPxJ4d2x9jPmU=")
	wgConfig := main.GenerateBaseConfig(bootstrapConfig, privateKey)

	expected := `[Interface]
PrivateKey = wFTAVe1stJPp0xQ+FE9so56uKh0jaHkPxJ4d2x9jPmU=

[Peer]
PublicKey = PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=
AllowedIPs = 10.1.1.2/32
Endpoint = 69.1.1.1:51820
`
	assert.Equal(t, expected, wgConfig)
}

func TestGenerateWireGuardPeers(t *testing.T) {
	gateways := []main.Gateway{{
		PublicKey: "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		Endpoint:  "13.37.13.37:51820",
		IP:        "10.255.240.2",
		Routes:    []string{"13.37.69.0/24", "13.37.59.69/32"},
	}}

	config := main.GenerateWireGuardPeers(gateways)
	expected := `[Peer]
PublicKey = PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=
AllowedIPs = 13.37.69.0/24,13.37.59.69/32,10.255.240.2/32
Endpoint = 13.37.13.37:51820
`
	assert.Equal(t, expected, config)
}

//func TestAuth(t *testing.T) {
//	redirectURL := "http://localhost:1337"
//
//	ctx := context.Background()
//	conf := &oauth2.Config{
//		ClientID: "5d69cfe1-b300-4a1a-95ec-4752d07ccab1",
//		//ClientSecret: "YOUR_CLIENT_SECRET",
//		Scopes:      []string{"openid", "5d69cfe1-b300-4a1a-95ec-4752d07ccab1/.default", "offline_access"},
//		Endpoint:    endpoints.AzureAD("62366534-1ec3-4962-8869-9b5535279d0b"),
//		RedirectURL: redirectURL,
//	}
//	server := &http.Server{Addr: ":1337"}
//
//	// Redirect user to consent page to ask for permission
//	// for the scopes specified above.
//
//	codeVerifier, err := go_oauth_pkce_code_verifier.CreateCodeVerifier()
//	if err != nil {
//		log.Fatalf("oopsie")
//	}
//
//	method := oauth2.SetAuthURLParam("code_challenge_method", "S256")
//	challenge := oauth2.SetAuthURLParam("code_challenge", codeVerifier.CodeChallengeS256())
//
//	url := conf.AuthCodeURL("rolled_dice_it_was_4", oauth2.AccessTypeOffline, method, challenge)
//
//	fmt.Printf("Visit the URL for the auth dialog: %v\n", url)
//
//	// Use the authorization code that is pushed to the redirect
//	// URL. Exchange will do the handshake to retrieve the
//	// initial access token. The HTTP Client returned by
//	// conf.Client will refresh the token as necessary.
//
//	// define a handler that will get the authorization code, call the token endpoint, and close the HTTP server
//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//		// get the authorization code
//		code := r.URL.Query().Get("code")
//		if code == "" {
//			fmt.Println("snap: Url Param 'code' is missing")
//			io.WriteString(w, "Error: could not find 'code' URL parameter\n")
//
//			return
//		}
//
//
//		codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", codeVerifier.String())
//		tok, err := conf.Exchange(ctx, code, codeVerifierParam)
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		io.WriteString(w, "hei\n")
//		fmt.Println("Successfully logged in.")
//
//		client := conf.Client(ctx, tok)
//		client.Get("...")
//	})
//
//	server.ListenAndServe()
//	defer server.Close()
//
//	assert.Equal(t, true, true)
//}
