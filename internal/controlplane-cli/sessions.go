package controlplanecli

import (
	"fmt"

	"github.com/nais/device/internal/pb"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func ListSessions(c *cli.Context) error {
	conn, err := grpc.NewClient(
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
		fmt.Printf("user: %s, lastSeen: %v, ipv4: %s, ipv6: %s, pubkey: %q, expired: %t, issues: %v\n",
			s.Device.GetUsername(),
			s.Device.GetLastSeen(),
			s.Device.GetIpv4(),
			s.Device.GetIpv6(),
			s.Device.GetPublicKey(),
			s.Expired(),
			len(s.Device.GetIssues()),
		)
	}

	return nil
}
