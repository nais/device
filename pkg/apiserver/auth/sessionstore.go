package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/pb"
)

var ErrNoSession = errors.New("no active session")

// sessionStore provides a database-backed session storage, with an in-memory caching layer.
type sessionStore struct {
	db    database.APIServer
	cache map[string]*pb.Session
	lock  sync.Mutex
}

type SessionStore interface {
	Get(context.Context, string) (*pb.Session, error)
	Set(context.Context, *pb.Session) error
	CachedSessionFromDeviceID(int64) (*pb.Session, error)
	All() []*pb.Session
}

func NewSessionStore(db database.APIServer) *sessionStore {
	return &sessionStore{
		db:    db,
		cache: make(map[string]*pb.Session),
	}
}

func (store *sessionStore) Get(ctx context.Context, key string) (*pb.Session, error) {
	store.lock.Lock()
	defer store.lock.Unlock()

	session, exists := store.cache[key]
	if exists && session != nil {
		return session, nil
	}

	session, err := store.db.ReadSessionInfo(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read session from database: %w", err)
	}

	store.cache[session.Key] = session

	return session, nil
}

func (store *sessionStore) Set(ctx context.Context, session *pb.Session) error {
	store.lock.Lock()
	defer store.lock.Unlock()

	err := store.db.AddSessionInfo(ctx, session)
	if err != nil {
		return fmt.Errorf("store session in database: %w", err)
	}

	store.cache[session.Key] = session
	return nil
}

func (store *sessionStore) Warmup(ctx context.Context) error {
	sessions, err := store.db.ReadSessionInfos(ctx)
	if err != nil {
		return fmt.Errorf("warm cache from database: %w", err)
	}

	store.lock.Lock()
	defer store.lock.Unlock()

	for _, session := range sessions {
		store.cache[session.Key] = session
	}

	return nil
}

func (store *sessionStore) CachedSessionFromDeviceID(deviceID int64) (*pb.Session, error) {
	store.lock.Lock()
	defer store.lock.Unlock()

	for _, session := range store.cache {
		if session.GetDevice().GetId() == deviceID {
			return session, nil
		}
	}

	return nil, fmt.Errorf("%w: device %d", ErrNoSession, deviceID)
}

func (store *sessionStore) All() []*pb.Session {
	store.lock.Lock()
	defer store.lock.Unlock()

	all := make([]*pb.Session, 0)
	for _, s := range store.cache {
		if s.Expired() {
			continue
		}

		all = append(all, s)
	}

	return all
}
