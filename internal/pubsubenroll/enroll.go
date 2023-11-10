package pubsubenroll

import "github.com/nais/device/internal/pb"

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
	WireGuardIPv4        string        `json:"wireguard_ip"` // TODO rename to wireguard_ipv4
	WireGuardIPv6        string        `json:"wireguard_ipv6"`
	Peers                []*pb.Gateway `json:"peers"`
}

const (
	TypeEnrollRequest  = "enroll-request"
	TypeEnrollResponse = "enroll-response"
)
