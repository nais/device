package controlplanecli

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/nais/device/pkg/device-agent/wireguard"
	"github.com/nais/device/pkg/passwordhash"
	"github.com/nais/device/pkg/pb"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

const FlagName = "name"
const FlagEndpoint = "endpoint"
const FlagAdminPassword = "admin-password"
const FlagPassword = "password"
const FlagPasswordHash = "password-hash"

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

func HashPassword(c *cli.Context) error {
	password := c.String(FlagPassword)

	salt, err := passwordhash.RandomBytes(16)
	if err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}

	key := passwordhash.HashPassword([]byte(password), salt)
	passhash := passwordhash.FormatHash(key, salt)

	fmt.Println(string(passhash))

	return nil
}

func EditGateway(c *cli.Context) error {
	conn, err := grpc.DialContext(
		c.Context,
		"127.0.0.1:8099",
		grpc.WithInsecure(),
	)

	if err != nil {
		return err
	}

	client := pb.NewAPIServerClient(conn)

	req := &pb.ModifyGatewayRequest{
		Password: c.String(FlagAdminPassword),
		Gateway: &pb.Gateway{
			Name: c.String(FlagName),
		},
	}

	gw, err := client.GetGateway(c.Context, req)
	if err != nil {
		return err
	}

	if gw == nil {
		return fmt.Errorf("gateway not found")
	}

	if c.IsSet(FlagPasswordHash) {
		gw.PasswordHash = c.String(FlagPasswordHash)
	}

	req.Gateway = gw

	_, err = client.UpdateGateway(c.Context, req)

	return err
}

func EnrollGateway(c *cli.Context) error {
	password, err := passwordhash.RandomBytes(32)
	if err != nil {
		return fmt.Errorf("generate password: %w", err)
	}

	salt, err := passwordhash.RandomBytes(16)
	if err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}

	key := passwordhash.HashPassword(password, salt)
	passhash := passwordhash.FormatHash(key, salt)

	privateKey := wireguard.WgGenKey()
	publicKey := wireguard.PublicKey(privateKey)

	req := &pb.ModifyGatewayRequest{
		Password: c.String(FlagAdminPassword),
		Gateway: &pb.Gateway{
			Name:         c.String(FlagName),
			PublicKey:    string(publicKey),
			Endpoint:     c.String(FlagEndpoint),
			PasswordHash: string(passhash),
		},
	}

	fmt.Fprintf(os.Stderr, "Enrolling a new gateway with API server:\n\n")
	fmt.Fprintf(os.Stderr, "name........: %s\n", req.Gateway.Name)
	fmt.Fprintf(os.Stderr, "publickey...: %s\n", req.Gateway.PublicKey)
	fmt.Fprintf(os.Stderr, "endpoint....: %s\n", req.Gateway.Endpoint)
	fmt.Fprintf(os.Stderr, "passhash....: %s\n", req.Gateway.PasswordHash)
	fmt.Fprintf(os.Stderr, "\n")

	conn, err := grpc.DialContext(
		c.Context,
		"127.0.0.1:8099",
		grpc.WithInsecure(),
	)

	if err != nil {
		return err
	}

	client := pb.NewAPIServerClient(conn)

	response, err := client.EnrollGateway(c.Context, req)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Gateway enrollment successful.\n")
	fmt.Fprintf(os.Stderr, "Please paste the following configuration into /etc/default/gateway-agent:\n\n")

	fmt.Printf("GATEWAY_AGENT_NAME=\"%s\"\n", response.GetGateway().GetName())
	fmt.Printf("GATEWAY_AGENT_APISERVERPASSWORD=\"%s\"\n", base64.StdEncoding.EncodeToString(password))
	fmt.Printf("GATEWAY_AGENT_PRIVATEKEY=\"%s\"\n", base64.StdEncoding.EncodeToString(privateKey))
	fmt.Printf("GATEWAY_AGENT_DEVICEIP=\"%s\"\n", response.GetGateway().GetIp())

	return err
}
