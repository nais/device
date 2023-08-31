package controlplanecli

import (
	"fmt"

	"github.com/nais/device/pkg/pb"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func ListSessions(c *cli.Context) error {
	conn, err := grpc.DialContext(
		c.Context,
		c.String(FlagAPIServer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
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
		fmt.Printf("user: %s, healthy: %t, ipv4: %s, ipv6: %s, pubkey: %q, expired: %t\n",
			s.Device.GetUsername(),
			s.Device.GetHealthy(),
			s.Device.GetIpv4(),
			s.Device.GetIpv6(),
			s.Device.GetPublicKey(),
			s.Expired(),
		)
	}

	return nil
}
