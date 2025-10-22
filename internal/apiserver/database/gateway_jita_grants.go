package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/nais/device/internal/apiserver/sqlc"
	"github.com/nais/device/internal/formats"
	"github.com/nais/device/pkg/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (db *database) GetGatewayJitaGrantsForUser(ctx context.Context, userID string) ([]*pb.GatewayJitaGrant, error) {
	rows, err := db.queries.GetGatewayJitaGrantsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	ret := make([]*pb.GatewayJitaGrant, len(rows))
	for i, row := range rows {
		var revoked *timestamppb.Timestamp
		if row.Revoked.Valid {
			revoked = timestamppb.New(stringToTime(row.Revoked.String))
		}

		ret[i] = &pb.GatewayJitaGrant{
			Id:      row.ID,
			Gateway: row.GatewayName,
			Created: timestamppb.New(stringToTime(row.Created)),
			Expires: timestamppb.New(stringToTime(row.Expires)),
			Revoked: revoked,
			Reason:  row.Reason,
		}
	}

	return ret, nil
}

func (db *database) UserHasAccessToPrivilegedGateway(ctx context.Context, userID, gatewayName string) (bool, error) {
	hasAccess, err := db.queries.UserHasAccessToPrivilegedGateway(ctx, sqlc.UserHasAccessToPrivilegedGatewayParams{
		UserID:      userID,
		GatewayName: gatewayName,
	})
	if err != nil {
		return false, err
	}

	return hasAccess == 1, nil
}

func (db *database) UsersWithAccessToPrivilegedGateway(ctx context.Context, gatewayName string) ([]string, error) {
	return db.queries.UsersWithAccessToPrivilegedGateway(ctx, gatewayName)
}

func (db *database) GrantPrivilegedGatewayAccess(ctx context.Context, userID, gatewayName string, expires time.Time, reason string) error {
	return db.queries.GrantPrivilegedGatewayAccess(ctx, sqlc.GrantPrivilegedGatewayAccessParams{
		UserID:      userID,
		GatewayName: gatewayName,
		Created:     time.Now().Format(formats.TimeFormat),
		Expires:     expires.Format(formats.TimeFormat),
		Reason:      reason,
	})
}

func (db *database) RevokePrivilegedGatewayAccess(ctx context.Context, userID, gatewayName string) error {
	return db.queries.RevokePrivilegedGatewayAccess(ctx, sqlc.RevokePrivilegedGatewayAccessParams{
		Revoked: sql.NullString{
			String: time.Now().Format(formats.TimeFormat),
			Valid:  true,
		},
		UserID:      userID,
		GatewayName: gatewayName,
	})
}
