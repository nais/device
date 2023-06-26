package controlplanecli

import (
	"fmt"

	"github.com/nais/device/pkg/pb"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

func ListSessions(c *cli.Context) error {
	conn, err := grpc.DialContext(
		c.Context,
		c.String(FlagAPIServer),
		grpc.WithInsecure(),
	)
	if err != nil {
		return err
	}

	client := pb.NewAPIServerClient(conn)
	resp, err := client.GetSessions(c.Context, &pb.GetSessionsRequest{
		Username: AdminUsername,
		Password: c.String(FlagAdminPassword),
	})
	if err != nil {
		return err
	}

	for _, s := range resp.GetSessions() {
		fmt.Printf("user: %s, healthy: %t, ip: %s, pubkey: %q, expired: %t\n",
			s.Device.GetUsername(),
			s.Device.GetHealthy(),
			s.Device.GetIp(),
			s.Device.GetPublicKey(),
			s.Expired(),
		)
	}

	return nil
}
