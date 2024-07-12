package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/pb"
)

var ErrNoSession = errors.New("no active session")

// sessionStore provides a database-backed session storage, with an in-memory caching layer.
type sessionStore struct {
	db database.Database

	byKey      map[string]*pb.Session
	byDeviceID map[int64]*pb.Session

	lock sync.Mutex
}

type SessionStore interface {
	Get(context.Context, string) (*pb.Session, error)
	Set(context.Context, *pb.Session) error
	All() []*pb.Session
	RefreshDevice(*pb.Device)
}

func NewSessionStore(db database.Database) *sessionStore {
	return &sessionStore{
		db:         db,
		byKey:      make(map[string]*pb.Session),
		byDeviceID: make(map[int64]*pb.Session),
	}
}

func (store *sessionStore) Get(ctx context.Context, key string) (*pb.Session, error) {
	store.lock.Lock()
	defer store.lock.Unlock()

	session, exists := store.byKey[key]
	if exists && session != nil {
		return session, nil
	}

	session, err := store.db.ReadSessionInfo(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read session from database: %w", err)
	}

	store.byKey[session.Key] = session
	store.byDeviceID[session.Device.Id] = session

	return session, nil
}

// Delete all sessions belonging to a specific device.
// The cache store MUST be locked before calling this function.
func (store *sessionStore) deleteSessionsForDeviceIDWithAssumedLock(deviceID int64) {
	session, exists := store.byDeviceID[deviceID]
	if !exists {
		return
	}

	delete(store.byKey, session.Key)
	delete(store.byDeviceID, deviceID)
}

func (store *sessionStore) Set(ctx context.Context, session *pb.Session) error {
	if session.GetDevice() == nil {
		return fmt.Errorf("store session in database: device info not given")
	}

	store.lock.Lock()
	defer store.lock.Unlock()

	store.deleteSessionsForDeviceIDWithAssumedLock(session.GetDevice().GetId())

	err := store.db.AddSessionInfo(ctx, session)
	if err != nil {
		return fmt.Errorf("store session in database: %w", err)
	}

	store.byDeviceID[session.Device.Id] = session
	store.byKey[session.Key] = session
	return nil
}

func (store *sessionStore) Warmup(ctx context.Context) error {
	err := store.db.RemoveExpiredSessions(ctx)
	if err != nil {
		return err
	}

	sessions, err := store.db.ReadSessionInfos(ctx)
	if err != nil {
		return fmt.Errorf("warm cache from database: %w", err)
	}

	store.lock.Lock()
	defer store.lock.Unlock()

	for _, session := range sessions {
		store.byKey[session.Key] = session
		store.byDeviceID[session.Device.Id] = session
	}

	return nil
}

func (store *sessionStore) All() []*pb.Session {
	store.lock.Lock()
	defer store.lock.Unlock()

	all := make([]*pb.Session, 0)
	for id, s := range store.byDeviceID {
		if s.Expired() {
			store.deleteSessionsForDeviceIDWithAssumedLock(id)
			continue
		}

		all = append(all, s)
	}

	return all
}

func (store *sessionStore) RefreshDevice(device *pb.Device) {
	store.lock.Lock()
	defer store.lock.Unlock()

	d, exists := store.byDeviceID[device.Id]
	if exists {
		d.Device = device
	}
}
