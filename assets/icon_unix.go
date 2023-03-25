//go:build linux || darwin

package assets

import _ "embed"

var (
	//go:embed nais-logo-blue.png
	NaisLogoBlue []byte
	//go:embed nais-logo-bw-connected.ico
	NaisLogoBwConnected []byte
	//go:embed nais-logo-bw-disconnected.ico
	NaisLogoBwDisconnected []byte
	//go:embed nais-logo-green.png
	NaisLogoGreen []byte
	//go:embed nais-logo-red.png
	NaisLogoRed []byte
	//go:embed nais-logo-yellow.png
	NaisLogoYellow []byte
)
