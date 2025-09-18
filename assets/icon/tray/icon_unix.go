//go:build linux || darwin

package tray

import _ "embed"

var (
	//go:embed unix/nais-logo-blue.png
	NaisLogoBlue []byte
	//go:embed unix/nais-logo-bw-connected.png
	NaisLogoBwConnected []byte
	//go:embed unix/nais-logo-bw-disconnected.png
	NaisLogoBwDisconnected []byte
	//go:embed unix/nais-logo-green.png
	NaisLogoGreen []byte
	//go:embed unix/nais-logo-red.png
	NaisLogoRed []byte
	//go:embed unix/nais-logo-yellow.png
	NaisLogoYellow []byte
)
