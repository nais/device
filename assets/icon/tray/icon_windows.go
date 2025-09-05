package tray

import _ "embed"

var (
	//go:embed windows/nais-logo-blue.ico
	NaisLogoBlue []byte
	//go:embed windows/nais-logo-bw-connected.ico
	NaisLogoBwConnected []byte
	//go:embed windows/nais-logo-bw-disconnected.ico
	NaisLogoBwDisconnected []byte
	//go:embed windows/nais-logo-green.ico
	NaisLogoGreen []byte
	//go:embed windows/nais-logo-red.ico
	NaisLogoRed []byte
	//go:embed windows/nais-logo-yellow.ico
	NaisLogoYellow []byte
)
