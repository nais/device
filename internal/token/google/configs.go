package google

import "github.com/nais/device/internal/token"

const (
	clientID           = "955023559628-g51n36t4icbd6lq7ils4r0ol9oo8kpk0.apps.googleusercontent.com"
	googleJwksEndpoint = "https://www.googleapis.com/oauth2/v3/certs"
	googleIssuer       = "https://accounts.google.com"
)

var APIServerConfig = token.Config{
	ClientID: clientID,
	Issuer:   googleIssuer,
	Endpoint: googleJwksEndpoint,
}
