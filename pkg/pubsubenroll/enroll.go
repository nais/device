package pubsubenroll

import "github.com/nais/device/pkg/pb"

type Response struct {
	APIServerGRPCAddress string        `json:"api_server_grpc_address"`
	WireGuardIP          string        `json:"wireguard_ip"`
	Peers                []*pb.Gateway `json:"peers"`
}

const (
	TypeEnrollRequest  = "enroll-request"
	TypeEnrollResponse = "enroll-response"
)
