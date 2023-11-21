//go:build linux || darwin
// +build linux darwin

package assets

import _ "embed"

var (
	//go:embed nais-logo-blue@96x96.png
	NaisLogoBlue []byte
	//go:embed nais-logo-bw-connected.png
	NaisLogoBwConnected []byte
	//go:embed nais-logo-bw-disconnected.png
	NaisLogoBwDisconnected []byte
	//go:embed nais-logo-green@96x96.png
	NaisLogoGreen []byte
	//go:embed nais-logo-red@96x96.png
	NaisLogoRed []byte
	//go:embed nais-logo-yellow@96x96.png
	NaisLogoYellow []byte
)
