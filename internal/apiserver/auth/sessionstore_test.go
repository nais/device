package auth_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSessionStore_SetAndGetFromCache(t *testing.T) {
	ctx := context.Background()
	db := database.NewMockDatabase(t)
	store := auth.NewSessionStore(db)

	session := &pb.Session{
		Key:    "abc",
		Device: &pb.Device{},
	}

	db.On("AddSessionInfo", mock.Anything, session).Return(nil).Once()

	err := store.Set(ctx, session)
	assert.NoError(t, err)

	retrieved, err := store.Get(ctx, session.Key)
	assert.EqualValues(t, session, retrieved)
	assert.NoError(t, err)
}

func TestSessionStore_GetFromDatabase(t *testing.T) {
	ctx := context.Background()
	db := database.NewMockDatabase(t)
	store := auth.NewSessionStore(db)

	session := &pb.Session{
		Key: "abc",
	}

	// Hit database only once
	db.On("ReadSessionInfo", mock.Anything, "abc").Return(session, nil).Once()

	// Retrieve uncached session from database
	retrieved, err := store.Get(ctx, "abc")
	assert.EqualValues(t, session, retrieved)
	assert.NoError(t, err)

	// Retrieve again, call shouldn't hit database
	retrieved, err = store.Get(ctx, "abc")
	assert.EqualValues(t, session, retrieved)
	assert.NoError(t, err)
}

func TestSessionStore_Errors(t *testing.T) {
	ctx := context.Background()
	db := database.NewMockDatabase(t)
	store := auth.NewSessionStore(db)

	session := &pb.Session{
		Key:    "abc",
		Device: &pb.Device{},
	}
	dbError := errors.New("error from database")

	// Return error from database layer
	db.On("ReadSessionInfo", mock.Anything, "abc").Return(nil, dbError).Once()
	db.On("AddSessionInfo", mock.Anything, session).Return(dbError).Once()
	db.On("ReadSessionInfos", mock.Anything).Return(nil, dbError).Once()
	db.On("RemoveExpiredSessions", mock.Anything).Return(nil).Once()

	// Retrieve uncached session from database
	retrieved, err := store.Get(ctx, "abc")
	assert.Nil(t, retrieved)
	assert.EqualError(t, err, "read session from database: error from database")

	// Persist to database
	err = store.Set(ctx, session)
	assert.EqualError(t, err, "store session in database: error from database")

	// Fail warmup
	err = store.Warmup(ctx)
	assert.EqualError(t, err, "warm cache from database: error from database")
}

func TestSessionStore_Warmup(t *testing.T) {
	ctx := context.Background()
	db := database.NewMockDatabase(t)
	store := auth.NewSessionStore(db)

	sessions := make([]*pb.Session, 20)
	for i := range sessions {
		sessions[i] = &pb.Session{
			Key: strconv.Itoa(i),
		}
	}

	// Return from database layer
	db.On("ReadSessionInfos", mock.Anything).Return(sessions, nil).Once()
	db.On("RemoveExpiredSessions", mock.Anything).Return(nil).Once()

	// Warmup cache
	err := store.Warmup(ctx)
	assert.NoError(t, err)

	// Retrieve cached session
	retrieved, err := store.Get(ctx, "14")
	assert.Equal(t, &pb.Session{Key: "14"}, retrieved)
	assert.NoError(t, err)
}

func TestSessionStore_UpdateDevice(t *testing.T) {
	ctx := context.Background()
	db := database.NewMockDatabase(t)
	store := auth.NewSessionStore(db)

	now := time.Now()
	sessions := make([]*pb.Session, 20)
	for i := range sessions {
		sessions[i] = &pb.Session{
			Key: strconv.Itoa(i),
			Device: &pb.Device{
				Id:       int64(i),
				LastSeen: timestamppb.New(now),
			},
		}
	}

	// Return from database layer
	db.On("ReadSessionInfos", mock.Anything).Return(sessions, nil).Once()
	db.On("RemoveExpiredSessions", mock.Anything).Return(nil).Once()

	// Warmup cache
	err := store.Warmup(ctx)
	assert.NoError(t, err)

	updatedDevice := &pb.Device{
		Id:       int64(0),
		LastSeen: timestamppb.New(now.Add(2 * time.Hour)),
	}

	sess, err := store.Get(ctx, "0")
	assert.NoError(t, err)
	assert.False(t, sess.GetDevice().GetLastSeen().AsTime().Equal(updatedDevice.GetLastSeen().AsTime()))

	store.UpdateDevice(updatedDevice)

	sess, err = store.Get(ctx, "0")
	assert.NoError(t, err)
	assert.True(t, sess.GetDevice().GetLastSeen().AsTime().Equal(updatedDevice.GetLastSeen().AsTime()))
}

// Test that existing sessions with the same device id are removed.
func TestSessionStore_ReplaceOnSet(t *testing.T) {
	ctx := context.Background()
	db := database.NewMockDatabase(t)
	store := auth.NewSessionStore(db)

	now := time.Now()

	// Return from database layer
	db.EXPECT().AddSessionInfo(mock.Anything, mock.Anything).Return(nil).Times(3)
	db.EXPECT().ReadSessionInfo(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("oops")).Once()

	assert.NoError(t, store.Set(ctx, &pb.Session{
		Key: "old_key_1",
		Device: &pb.Device{
			Id:       123,
			LastSeen: timestamppb.New(now),
			Serial:   "old",
		},
	}))

	assert.NoError(t, store.Set(ctx, &pb.Session{
		Key: "old_key_2",
		Device: &pb.Device{
			Id:       456,
			LastSeen: timestamppb.New(now),
			Serial:   "old",
		},
	}))

	// Assert that the device is stored
	session, _ := store.Get(ctx, "old_key_1")
	assert.Equal(t, session.GetDevice().GetId(), int64(123))
	assert.Equal(t, session.GetDevice().GetSerial(), "old")

	assert.NoError(t, store.Set(ctx, &pb.Session{
		Key: "new_key_1",
		Device: &pb.Device{
			Id:       123,
			LastSeen: timestamppb.New(now),
			Serial:   "new",
		},
	}))

	// Assert that the old key is deleted and the new key refers to the correct device
	_, err := store.Get(ctx, "old_key_1")
	assert.Error(t, err)
	session, err = store.Get(ctx, "new_key_1")
	assert.NoError(t, err)
	assert.Equal(t, session.GetDevice().GetId(), int64(123))
	assert.Equal(t, session.GetDevice().GetSerial(), "new")

	// Assert that the other device still exists in its original state
	session, _ = store.Get(ctx, "old_key_2")
	assert.Equal(t, session.GetDevice().GetId(), int64(456))
	assert.Equal(t, session.GetDevice().GetSerial(), "old")

}
