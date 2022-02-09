package pb_test

import (
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
	afterDeadline := now.Add(-25 * time.Hour)

	deviceSeenRecently := &pb.Device{
		KolideLastSeen: timestamppb.New(withinDeadline),
	}

	deviceNotSeenRecently := &pb.Device{
		KolideLastSeen: timestamppb.New(afterDeadline),
	}

	assert.True(t, deviceSeenRecently.KolideSeenRecently())
	assert.False(t, deviceNotSeenRecently.KolideSeenRecently())
}
