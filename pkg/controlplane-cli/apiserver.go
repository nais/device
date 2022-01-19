package controlplanecli

import (
	"errors"
	"fmt"
	"io"

	"github.com/nais/device/pkg/pb"
	cli "github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

const AdminPasswordFlagName = "admin_password"

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
	stream, err := client.AdminListGateways(c.Context, &pb.AdminListGatewayRequest{
		Password: c.String(AdminPasswordFlagName),
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
