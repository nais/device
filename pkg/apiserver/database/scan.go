package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/pb"
)

type Scanner interface {
	Scan(...any) error
}

const DeviceFields = "id, serial, username, psk, platform, last_updated, kolide_last_seen, healthy, public_key, ip"
const GatewayFields = "public_key, access_group_ids, endpoint, ip, routes, name, requires_privileged_access, password_hash"
const sessionFields = "key, expiry, device_id, groups, object_id"

func scanGateway(row Scanner) (*pb.Gateway, error) {
	gateway := &pb.Gateway{}
	var routes string
	var accessGroupIDs string
	var passhash *string

	err := row.Scan(&gateway.PublicKey, &accessGroupIDs, &gateway.Endpoint, &gateway.Ip, &routes, &gateway.Name, &gateway.RequiresPrivilegedAccess, &passhash)
	if err != nil {
		return nil, fmt.Errorf("scanning gateway: %w", err)
	}

	if passhash != nil {
		gateway.PasswordHash = *passhash
	}

	if len(accessGroupIDs) != 0 {
		gateway.AccessGroupIDs = strings.Split(accessGroupIDs, ",")
	}

	if len(routes) != 0 {
		gateway.Routes = strings.Split(routes, ",")
	}

	return gateway, nil
}

func scanDevice(row Scanner) (*pb.Device, error) {
	var lastUpdated *time.Time
	var kolideLastSeen *time.Time

	device := &pb.Device{}

	err := row.Scan(&device.Id, &device.Serial, &device.Username, &device.Psk, &device.Platform, &lastUpdated, &kolideLastSeen, &device.Healthy, &device.PublicKey, &device.Ip)

	if err != nil {
		return nil, fmt.Errorf("scan device row: %s", err)
	}

	if lastUpdated != nil {
		device.LastUpdated = timestamppb.New(*lastUpdated)
	}
	if kolideLastSeen != nil {
		device.KolideLastSeen = timestamppb.New(*kolideLastSeen)
	}

	return device, nil
}

func scanSession(row Scanner) (*pb.Session, error) {
	session := &pb.Session{}

	var groups string
	var deviceID int64
	var expiry *time.Time

	err := row.Scan(&session.Key, &expiry, &deviceID, &groups, &session.ObjectID)
	if err != nil {
		return nil, fmt.Errorf("scan session row: %w", err)
	}

	session.Groups = strings.Split(groups, ",")
	session.Device = &pb.Device{Id: deviceID}
	if expiry != nil {
		session.Expiry = timestamppb.New(*expiry)
	}

	return session, nil
}

func scanSessionWithDevice(ctx context.Context, row Scanner, db APIServer) (*pb.Session, error) {
	session, err := scanSession(row)
	if err != nil {
		return nil, err
	}

	session.Device, err = db.ReadDeviceById(ctx, session.Device.Id)

	if err != nil {
		return nil, fmt.Errorf("read device: %w", err)
	}

	return session, nil
}
