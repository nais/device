package auth_test

import (
	"context"
	"errors"
	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/mocks"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"strconv"
	"testing"
)

func TestSessionStore_SetAndGetFromCache(t *testing.T) {
	ctx := context.Background()
	db := &mocks.APIServer{}
	store := auth.NewSessionStore(db)

	session := &pb.Session{
		Key: "abc",
	}

	db.On("AddSessionInfo", mock.Anything, session).Return(nil).Once()

	err := store.Set(ctx, session)
	assert.NoError(t, err)

	retrieved, err := store.Get(ctx, session.Key)
	assert.EqualValues(t, session, retrieved)
	assert.NoError(t, err)

	// Assert that our query hit the cache, not the database
	db.AssertExpectations(t)
}

func TestSessionStore_GetFromDatabase(t *testing.T) {
	ctx := context.Background()
	db := &mocks.APIServer{}
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

	db.AssertExpectations(t)
}

func TestSessionStore_Errors(t *testing.T) {
	ctx := context.Background()
	db := &mocks.APIServer{}
	store := auth.NewSessionStore(db)

	session := &pb.Session{
		Key: "abc",
	}
	dbError := errors.New("error from database")

	// Return error from database layer
	db.On("ReadSessionInfo", mock.Anything, "abc").Return(nil, dbError).Once()
	db.On("AddSessionInfo", mock.Anything, session).Return(dbError).Once()
	db.On("ReadSessionInfos", mock.Anything).Return(nil, dbError).Once()

	// Retrieve uncached session from database
	retrieved, err := store.Get(ctx, "abc")
	assert.Nil(t, retrieved)
	assert.EqualError(t, err, "read session from database: error from database")

	// Persist to database
	err = store.Set(ctx, session)
	assert.EqualError(t, err, "store session in database: error from database")

	// Get cached device
	session, err = store.CachedSessionFromDeviceID(14)
	assert.EqualError(t, err, "no active session for device 14")
	assert.Nil(t, session)

	// Fail warmup
	err = store.Warmup(ctx)
	assert.EqualError(t, err, "warm cache from database: error from database")

	db.AssertExpectations(t)
}

func TestSessionStore_Warmup(t *testing.T) {
	ctx := context.Background()
	db := &mocks.APIServer{}
	store := auth.NewSessionStore(db)

	sessions := make([]*pb.Session, 20)
	for i := range sessions {
		sessions[i] = &pb.Session{
			Key: strconv.Itoa(i),
		}
	}

	// Return from database layer
	db.On("ReadSessionInfos", mock.Anything).Return(sessions, nil).Once()

	// Warmup cache
	err := store.Warmup(ctx)
	assert.NoError(t, err)

	// Retrieve cached session
	retrieved, err := store.Get(ctx, "14")
	assert.Equal(t, &pb.Session{Key: "14"}, retrieved)
	assert.NoError(t, err)

	db.AssertExpectations(t)
}

func TestSessionStore_CachedSessionFromDeviceID(t *testing.T) {
	ctx := context.Background()
	db := &mocks.APIServer{}
	store := auth.NewSessionStore(db)

	sessions := make([]*pb.Session, 20)
	for i := range sessions {
		sessions[i] = &pb.Session{
			Key: strconv.Itoa(i),
			Device: &pb.Device{
				Id: int64(i),
			},
		}
	}

	// Return from database layer
	db.On("ReadSessionInfos", mock.Anything).Return(sessions, nil).Once()

	// Warmup cache
	err := store.Warmup(ctx)
	assert.NoError(t, err)

	// Retrieve cached session
	expected := &pb.Session{
		Key: "14",
		Device: &pb.Device{
			Id: 14,
		},
	}
	retrieved, err := store.CachedSessionFromDeviceID(14)
	assert.Equal(t, expected, retrieved)
	assert.NoError(t, err)

	db.AssertExpectations(t)
}
