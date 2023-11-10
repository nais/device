package auth

import (
	"time"

	"github.com/nais/device/internal/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type LegacySessionInfo struct {
	Key    string `json:"key"`
	Expiry int64  `json:"expiry"`
}

func (s *LegacySessionInfo) ToProtobuf() *pb.Session {
	return &pb.Session{
		Key:    s.Key,
		Expiry: timestamppb.New(time.Unix(s.Expiry, 0)),
	}
}

func LegacySessionFromProtobuf(s *pb.Session) *LegacySessionInfo {
	return &LegacySessionInfo{
		Key:    s.GetKey(),
		Expiry: s.GetExpiry().AsTime().Unix(),
	}
}
