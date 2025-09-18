package enroller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/enroll"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
)

type localEnroller struct {
	db             database.Database
	log            logrus.FieldLogger
	enrollmentsUrl string
}

func NewLocalEnroll(db database.Database, enrollmentsUrl string) localEnroller {
	if enrollmentsUrl == "" {
		enrollmentsUrl = " http://localhost:8081/enrollments"
	}
	return localEnroller{
		db:             db,
		log:            logrus.WithField("component", "local_enroller"),
		enrollmentsUrl: enrollmentsUrl,
	}
}

var _ AutoEnroller = &localEnroller{}

func (l *localEnroller) Run(ctx context.Context) error {
	l.log.Info("local enroller: running")
	for {
		select {
		case <-ctx.Done():
			l.log.Info("local enroller: shutting down")
			return nil
		case <-time.After(500 * time.Millisecond):
			enrollments, err := l.getEnrollments(ctx)
			if err != nil {
				l.log.WithError(err).Error("local enroller: error getting enrollments")
				continue
			}
			for _, enrollment := range enrollments {
				l.enroll(ctx, enrollment)
			}
		}
	}
}

func (l *localEnroller) getEnrollments(ctx context.Context) ([]enroll.DeviceRequest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.enrollmentsUrl, nil)
	if err != nil {
		msg := "local enroller: error creating request"
		l.log.WithError(err).Error(msg)
		return nil, fmt.Errorf("%s", msg)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		msg := "local enroller: error fetching enrollments"
		l.log.WithError(err).Error(msg)
		return nil, fmt.Errorf("%s", msg)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := "local enroller: non-200 response fetching enrollments"
		l.log.WithField("status", resp.StatusCode).Error(msg)
		return nil, fmt.Errorf("%s", msg)
	}

	var enrollments []enroll.DeviceRequest
	if err := json.NewDecoder(resp.Body).Decode(&enrollments); err != nil {
		msg := "local enroller: error decoding enrollments"
		l.log.WithError(err).Error(msg)
		return nil, fmt.Errorf("%s", msg)
	}

	return enrollments, nil
}

func (l *localEnroller) enroll(ctx context.Context, enrollment enroll.DeviceRequest) error {
	device := &pb.Device{
		Username:  enrollment.Owner,
		Serial:    enrollment.Serial,
		Platform:  enrollment.Platform,
		PublicKey: string(enrollment.WireGuardPublicKey),
	}

	if err := l.db.AddDevice(ctx, device); err != nil {
		msg := "local enroller: error adding device to database"
		l.log.WithError(err).Error(msg)
		l.log.Infof("continuing enroll, as this is normal for local enrollments: %v", err)
	}
	payload, err := json.Marshal(&enroll.Response{
		APIServerGRPCAddress: "localhost:8099",
		WireGuardIPv4:        "127.0.0.69",
		Peers: []*pb.Gateway{
			{
				Name: "apiserver",
				Ipv4: "127.0.0.1",
			},
		},
	})
	if err != nil {
		msg := "local enroller: error marshalling enrollment response"
		l.log.WithError(err).Error(msg)
		return fmt.Errorf("%s", msg)
	}

	httpResp, err := http.NewRequestWithContext(ctx, http.MethodPost, l.enrollmentsUrl, bytes.NewBuffer(payload))
	if err != nil {
		msg := "local enroller: error creating response request"
		l.log.WithError(err).Error(msg)
		return fmt.Errorf("%s", msg)
	}

	resp, err := http.DefaultClient.Do(httpResp)
	if err != nil {
		msg := "local enroller: error sending enrollment response"
		l.log.WithError(err).Error(msg)
		return fmt.Errorf("%s", msg)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg := "local enroller: non-200 response sending enrollment response"
		l.log.WithField("status", resp.StatusCode).Error(msg)
		return fmt.Errorf("%s", msg)
	}

	l.log.WithField("enrollment", enrollment).Info("local enroller: successfully added device to database")
	return nil
}
