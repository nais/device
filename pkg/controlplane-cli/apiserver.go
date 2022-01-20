package controlplanecli

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

const FlagAdminPassword = "admin-password"
const FlagRequest = "request"

func ListGateways(c *cli.Context) error {
	conn, err := grpc.DialContext(
		c.Context,
		"127.0.0.1:8099",
		grpc.WithInsecure(),
	)
	if err != nil {
		return err
	}

	client := pb.NewAPIServerClient(conn)
	stream, err := client.ListGateways(c.Context, &pb.ListGatewayRequest{
		Password: c.String(FlagAdminPassword),
	})
	if err != nil {
		return err
	}
	for stream.Context().Err() == nil {
		gw, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		fmt.Println(gw)
	}

	return stream.Context().Err()
}

func EnrollGateway(c *cli.Context) error {
	conn, err := grpc.DialContext(
		c.Context,
		"127.0.0.1:8099",
		grpc.WithInsecure(),
	)
	if err != nil {
		return err
	}

	client := pb.NewAPIServerClient(conn)

	req := &pb.EnrollGatewayRequest{}
	buf, err := base64.StdEncoding.DecodeString(c.String(FlagRequest))
	if err != nil {
		return fmt.Errorf("error in base64 string: %w", err)
	}

	err = proto.Unmarshal(buf, req)
	if err != nil {
		return fmt.Errorf("error in decoded protobuf string: %w", err)
	}

	log.Infof("Enrolling gateway '%s' with endpoint '%s' and public key '%s'", req.GetGateway().GetName(), req.GetGateway().GetIp(), req.GetGateway().GetPublicKey())

	req.Password = c.String(FlagAdminPassword)

	_, err = client.EnrollGateway(c.Context, req)
	if err != nil {
		return err
	}

	return err
}
