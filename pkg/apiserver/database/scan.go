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
	Scan(...interface{}) error
}

const DeviceFields = "id, serial, username, psk, platform, last_updated, kolide_last_seen, healthy, public_key, ip"

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
