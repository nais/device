package pubsubenroll

import "github.com/nais/device/pkg/pb"

type DeviceRequest struct {
	Platform           string `json:"platform"`
	Owner              string `json:"owner"`
	Serial             string `json:"serial"`
	WireGuardPublicKey []byte `json:"wireguard_public_key"`
}

type GatewayRequest struct {
	WireGuardPublicKey []byte `json:"wireguard_public_key"`
	Name               string `json:"name"`
	Endpoint           string `json:"endpoint"`
	HashedPassword     string `json:"hashed_password"`
}

type Response struct {
	APIServerGRPCAddress string        `json:"api_server_grpc_address"`
	WireGuardIP          string        `json:"wireguard_ip"`
	Peers                []*pb.Gateway `json:"peers"`
}

const (
	TypeEnrollRequest  = "enroll-request"
	TypeEnrollResponse = "enroll-response"
)
