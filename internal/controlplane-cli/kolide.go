package controlplanecli

import (
	"encoding/json"
	"os"

	"github.com/nais/device/pkg/pb"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetKolideCache(c *cli.Context) error {
	conn, err := grpc.NewClient(
		c.String(FlagAPIServer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}

	client := pb.NewAPIServerClient(conn)
	resp, err := client.GetKolideCache(c.Context, &pb.GetKolideCacheRequest{
		Username: AdminUsername,
		Password: c.String(FlagAdminPassword),
	})
	if err != nil {
		return err
	}

	out := struct {
		Devices json.RawMessage
		Checks  json.RawMessage
	}{
		Checks: resp.RawChecks,
	}

	return json.NewEncoder(os.Stdout).Encode(out)
}
