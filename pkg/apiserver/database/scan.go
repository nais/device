package database

import (
	"context"
	"fmt"
	"github.com/nais/device/pkg/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strings"
	"time"
)

type Scanner interface {
	Scan(...interface{}) error
}

func scanDevice(row Scanner) (*pb.Device, error) {
	var lastUpdated *int64
	var kolideLastSeen *int64

	device := &pb.Device{}

	err := row.Scan(&device.Id, &device.Serial, &device.Username, &device.Psk, &device.Platform, &lastUpdated, &kolideLastSeen, &device.Healthy, &device.PublicKey, &device.Ip)

	if err != nil {
		return nil, fmt.Errorf("scan device row: %s", err)
	}

	if lastUpdated != nil {
		device.LastUpdated = timestamppb.New(time.Unix(*lastUpdated, 0))
	}
	if kolideLastSeen != nil {
		device.KolideLastSeen = timestamppb.New(time.Unix(*kolideLastSeen, 0))
	}

	return device, nil
}

func scanSession(row Scanner) (*pb.Session, error) {
	session := &pb.Session{}

	var groups string
	var deviceID int64
	var expiry *int64

	err := row.Scan(&session.Key, &expiry, &deviceID, &groups, &session.ObjectID)
	if err != nil {
		return nil, fmt.Errorf("scan session row: %w", err)
	}

	session.Groups = strings.Split(groups, ",")
	session.Device = &pb.Device{Id: deviceID}
	if expiry != nil {
		session.Expiry = timestamppb.New(time.Unix(*expiry, 0))
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