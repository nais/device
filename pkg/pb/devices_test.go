package pb_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestKolideSeenRecently(t *testing.T) {
	//now := time.Date(2069, 4, 20, 13, 37, 0, 0, nil)
	now := time.Now()
	withinDeadline := now.Add(-30 * time.Minute)
	afterDeadline := now.Add(-61 * time.Minute)

	deviceSeenRecently := &pb.Device{
		KolideLastSeen: timestamppb.New(withinDeadline),
	}

	deviceNotSeenRecently := &pb.Device{
		KolideLastSeen: timestamppb.New(afterDeadline),
	}

	assert.True(t, deviceSeenRecently.KolideSeenRecently())
	assert.False(t, deviceNotSeenRecently.KolideSeenRecently())
}

func TestWriteDevicePeerConfig(t *testing.T) {
	device := &pb.Device{
		PublicKey: "pub",
		Ip:        "ip",
	}

	buf := new(bytes.Buffer)
	err := device.WritePeerConfig(buf)
	assert.NoError(t, err)

	expected := `[Peer]
PublicKey = pub
AllowedIPs = ip

`
	assert.Equal(t, expected, buf.String())
}
