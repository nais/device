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
		want   bool
	}{
		{
			name: "healthy device",
			device: &kolideclient.Device{
				LastSeenAt: timep(time.Now()),
				Failures:   nil,
			},
			want: true,
		},
		{
			name: "unhealthy device (after grace period), but ignored failure",
			device: &kolideclient.Device{
				LastSeenAt: timep(time.Now()),
				Failures: []*kolideclient.DeviceFailure{
					{
						Timestamp:  timep(time.Now().Add(-2 * time.Hour)),
						ResolvedAt: nil,
						Ignored:    true,
						Check: &kolideclient.Check{
							Tags: []string{"DANGER"},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "unhealthy device (before grace period)",
			device: &kolideclient.Device{
				LastSeenAt: timep(time.Now()),
				Failures: []*kolideclient.DeviceFailure{
					{
						Timestamp:  timep(time.Now().Add(-30 * time.Minute)),
						ResolvedAt: nil,
						Ignored:    false,
						Check: &kolideclient.Check{
							Tags: []string{"DANGER"},
						},
					},
				},
			},
			want: true,
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
			want: false,
		},
		{
			name: "unhealthy device (danger failure without timestamp)",
			device: &kolideclient.Device{
				LastSeenAt: timep(time.Now()),
				Failures: []*kolideclient.DeviceFailure{
					{
						Timestamp:  nil,
						ResolvedAt: nil,
						Ignored:    false,
						Check: &kolideclient.Check{
							Tags: []string{"DANGER"},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "unhealthy device (multiple tags, use most severe)",
			device: &kolideclient.Device{
				LastSeenAt: timep(time.Now()),
				Failures: []*kolideclient.DeviceFailure{
					{
						Timestamp:  timep(time.Now().Add(-3 * time.Hour)),
						ResolvedAt: nil,
						Ignored:    false,
						Check: &kolideclient.Check{
							Tags: []string{"DANGER", "INFO"},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "healthy device (failure resolved)",
			device: &kolideclient.Device{
				LastSeenAt: timep(time.Now()),
				Failures: []*kolideclient.DeviceFailure{
					{
						Timestamp:  timep(time.Now().Add(-3 * time.Hour)),
						ResolvedAt: timep(time.Now()),
						Ignored:    false,
						Check: &kolideclient.Check{
							Tags: []string{"DANGER", "INFO"},
						},
					},
				},
			},
			want: true,
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

func timep(t time.Time) *time.Time {
	return &t
}
