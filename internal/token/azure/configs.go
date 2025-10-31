package azure

import (
	"fmt"

	"github.com/nais/device/internal/token"
)

const (
	azureTenantNav    = "62366534-1ec3-4962-8869-9b5535279d0b"
	apiServerClientID = "6e45010d-2637-4a40-b91d-d4cbb451fb57"
	jitaClientID      = "8b625469-1988-4adf-b02f-115315596ab8"
)

var (
	APIServerConfig = token.Config{
		ClientID: apiServerClientID,
		Issuer:   issuer(azureTenantNav),
		Endpoint: jwksEndpoint(azureTenantNav),
	}

	JITAConfig = token.Config{
		ClientID: jitaClientID,
		Issuer:   stsIssuer(azureTenantNav),
		Endpoint: jwksEndpointWithAppID(azureTenantNav, jitaClientID),
	}
)

func jwksEndpoint(tenant string) string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/keys", tenant)
}

func issuer(tenant string) string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenant)
}

func jwksEndpointWithAppID(tenant, appID string) string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/keys?appid=%s", tenant, appID)
}

func stsIssuer(tenant string) string {
	return fmt.Sprintf("https://sts.windows.net/%s/", tenant)
}
