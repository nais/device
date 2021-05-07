package kolide

import (
	kolideclient "github.com/nais/device/pkg/kolide-client"
	"reflect"
	"testing"
	"time"
)

func TestDeviceHealthy(t *testing.T) {
	tests := []struct {
		name   string
		device *kolideclient.Device
		want   *bool
	}{
		// ignored failure
		// failure already resolved
		{
			name: "healthy device",
			device: &kolideclient.Device{
				LastSeenAt: timep(time.Now()),
				Failures:   nil,
			},
			want: boolp(true),
		},
		{
			name: "unhealthy device (after grace period)",
			device: &kolideclient.Device{
				LastSeenAt: timep(time.Now()),
				Failures: []*kolideclient.DeviceFailure{
					{
						Timestamp:  timep(time.Now().Add(-2 * time.Hour)),
						ResolvedAt: nil,
						Ignored:    false,
						Check: &kolideclient.Check{
							Tags: []string{"DANGER"},
						},
					},
				},
			},
			want: boolp(true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeviceHealthy(tt.device); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeviceHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func boolp(b bool) *bool {
	return &b
}

func timep(t time.Time) *time.Time {
	return &t
}
