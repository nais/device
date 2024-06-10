package controlplanecli

import (
	"encoding/json"
	"os"

	"github.com/nais/device/internal/pb"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetKolideCache(c *cli.Context) error {
	conn, err := grpc.DialContext(
		c.Context,
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
		Devices: resp.RawDevices,
		Checks:  resp.RawChecks,
	}

	return json.NewEncoder(os.Stdout).Encode(out)
}
