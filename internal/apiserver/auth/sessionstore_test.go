package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/testdatabase"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSessionStore_SetAndGetFromCache(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Setup(t, false)
	store := auth.NewSessionStore(db)

	session := &pb.Session{
		Key:      "abc",
		Groups:   []string{"group1", "group2"},
		Expiry:   timestamppb.New(time.Now().Add(2 * time.Hour)),
		ObjectID: "oid",
		Device: &pb.Device{
			Id:       1,
			Platform: "linux",
		},
	}

	session2 := &pb.Session{
		Key:      "def",
		Groups:   []string{"group1", "group2"},
		Expiry:   timestamppb.New(time.Now().Add(2 * time.Hour)),
		ObjectID: "oid",
		Device: &pb.Device{
			Id:       1,
			Platform: "linux",
		},
	}

	device := &pb.Device{
		Serial:   "device-1",
		Platform: "linux",
		LastSeen: timestamppb.Now(),
	}
	if err := db.AddDevice(ctx, device); err != nil {
		t.Fatal(err)
	}

	err := store.Set(ctx, session)
	assert.NoError(t, err)

	err = store.Set(ctx, session2)
	assert.NoError(t, err)

	_, err = store.Get(ctx, session2.Key)
	// assert.EqualValues(t, session, retrieved)
	assert.NoError(t, err)
}

func TestSessionStore_Errors(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Setup(t, false)
	store := auth.NewSessionStore(db)

	session := &pb.Session{
		Key: "abc",
	}

	// Retrieve uncached session from database
	retrieved, err := store.Get(ctx, "abc")
	assert.Nil(t, retrieved)
	assert.EqualError(t, err, "read session from database: sql: no rows in result set")

	// Persist to database
	err = store.Set(ctx, session)
	assert.EqualError(t, err, "store session in database: device info not given")
}

func TestSessionStore_Warmup(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Setup(t, false)
	store := auth.NewSessionStore(db)

	for i := range 20 {
		deviceID := int64(i + 1)
		device := &pb.Device{
			Serial:    fmt.Sprintf("device-%v", deviceID),
			PublicKey: fmt.Sprintf("device-%v", deviceID),
			Platform:  "linux",
			LastSeen:  timestamppb.Now(),
		}
		if err := db.AddDevice(ctx, device); err != nil {
			t.Fatal(err)
		}

		session := &pb.Session{
			Key:    fmt.Sprintf("session-for-device-%v", deviceID),
			Expiry: timestamppb.New(time.Now().Add(2 * time.Hour)),
			Device: &pb.Device{
				Id: deviceID,
			},
		}
		if err := db.AddSessionInfo(ctx, session); err != nil {
			t.Fatal(err)
		}
	}

	// Warmup cache
	err := store.Warmup(ctx)
	assert.NoError(t, err)

	// Retrieve cached session
	retrieved, err := store.Get(ctx, fmt.Sprintf("session-for-device-%v", int64(14)))
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, int64(14), retrieved.GetDevice().GetId())
}

func TestSessionStore_UpdateDevice(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Setup(t, false)
	store := auth.NewSessionStore(db)

	now := time.Now()
	for i := range 20 {
		deviceID := int64(i + 1)
		device := &pb.Device{
			Serial:    fmt.Sprintf("device-%v", deviceID),
			PublicKey: fmt.Sprintf("device-%v", deviceID),
			Platform:  "linux",
			LastSeen:  timestamppb.New(now),
		}
		session := &pb.Session{
			Key: fmt.Sprintf("session-for-device-%v", deviceID),
			Device: &pb.Device{
				Id:       deviceID,
				LastSeen: timestamppb.New(now),
			},
		}

		if err := db.AddDevice(ctx, device); err != nil {
			t.Fatal(err)
		}
		if err := db.AddSessionInfo(ctx, session); err != nil {
			t.Fatal(err)
		}
	}

	updatedDevice := &pb.Device{
		Id:       int64(2),
		LastSeen: timestamppb.New(now.Add(2 * time.Hour)),
	}

	sess, err := store.Get(ctx, "session-for-device-2")
	assert.NoError(t, err)
	assert.False(t, sess.GetDevice().GetLastSeen().AsTime().Equal(updatedDevice.GetLastSeen().AsTime()))

	store.RefreshDevice(updatedDevice)

	sess, err = store.Get(ctx, "session-for-device-2")
	assert.NoError(t, err)
	assert.True(t, sess.GetDevice().GetLastSeen().AsTime().Equal(updatedDevice.GetLastSeen().AsTime()))
}

// Test that existing sessions with the same device id are removed.
func TestSessionStore_ReplaceOnSet(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Setup(t, false)
	store := auth.NewSessionStore(db)

	now := time.Now()

	device := &pb.Device{
		Serial:    "device-1",
		PublicKey: "device-1",
		Platform:  "linux",
		LastSeen:  timestamppb.New(now),
	}
	session := &pb.Session{
		Key:    "old_key_1",
		Device: &pb.Device{Id: 1, LastSeen: timestamppb.New(now)},
	}
	if err := db.AddDevice(ctx, device); err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, store.Set(ctx, session))

	device2 := &pb.Device{
		Serial:    "device-2",
		PublicKey: "device-2",
		Platform:  "linux",
		LastSeen:  timestamppb.Now(),
	}
	if err := db.AddDevice(ctx, device2); err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, store.Set(ctx, &pb.Session{
		Key: "old_key_2",
		Device: &pb.Device{
			Id:       2,
			LastSeen: timestamppb.New(now),
		},
	}))

	// Assert that the device is stored
	session, err := store.Get(ctx, "old_key_1")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), session.GetDevice().GetId())
	assert.Equal(t, "old_key_1", session.GetKey())

	assert.NoError(t, store.Set(ctx, &pb.Session{
		Key: "new_key_1",
		Device: &pb.Device{
			Id:       1,
			LastSeen: timestamppb.New(now),
		},
	}))

	// Assert that the old key is deleted and the new key refers to the correct device
	oldSession, err := store.Get(ctx, "old_key_1")
	assert.Error(t, err)
	assert.Nil(t, oldSession)
	session, err = store.Get(ctx, "new_key_1")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), session.GetDevice().GetId())
	assert.Equal(t, "new_key_1", session.GetKey())

	// Assert that the other device still exists in its original state
	session, err = store.Get(ctx, "old_key_2")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), session.GetDevice().GetId())
	assert.Equal(t, "old_key_2", session.GetKey())
}
