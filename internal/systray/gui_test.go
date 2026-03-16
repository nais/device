package systray

import (
	"testing"
	"time"

	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestReadyForConnectedIcon(t *testing.T) {
	connectedAt := timestamppb.New(time.Now())

	tests := []struct {
		name   string
		status *pb.AgentStatus
		want   bool
	}{
		{
			name:   "nil status",
			status: nil,
			want:   false,
		},
		{
			name: "missing connectedSince",
			status: &pb.AgentStatus{
				ConnectionState: pb.AgentState_Connected,
			},
			want: false,
		},
		{
			name: "all non-jita healthy",
			status: &pb.AgentStatus{
				ConnectionState: pb.AgentState_Connected,
				ConnectedSince:  connectedAt,
				Gateways: []*pb.Gateway{
					{Healthy: true},
					{Healthy: true},
				},
			},
			want: true,
		},
		{
			name: "non-jita unhealthy",
			status: &pb.AgentStatus{
				ConnectionState: pb.AgentState_Connected,
				ConnectedSince:  connectedAt,
				Gateways: []*pb.Gateway{
					{Healthy: true},
					{Healthy: false},
				},
			},
			want: false,
		},
		{
			name: "unhealthy jita ignored",
			status: &pb.AgentStatus{
				ConnectionState: pb.AgentState_Connected,
				ConnectedSince:  connectedAt,
				Gateways: []*pb.Gateway{
					{Healthy: true},
					{Healthy: false, RequiresPrivilegedAccess: true},
				},
			},
			want: true,
		},
		{
			name: "only jita gateways",
			status: &pb.AgentStatus{
				ConnectionState: pb.AgentState_Connected,
				ConnectedSince:  connectedAt,
				Gateways: []*pb.Gateway{
					{Healthy: false, RequiresPrivilegedAccess: true},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readyForConnectedIcon(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}
