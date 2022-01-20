package api

import (
	"context"

	"github.com/nais/device/pkg/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *grpcServer) EnrollGateway(ctx context.Context, r *pb.EnrollGatewayRequest) (*pb.EnrollGatewayResponse, error) {
	err := s.apikeyAuthenticator.Authenticate(AdminUsername, r.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	}

	gw := r.GetGateway()
	if gw == nil {
		return nil, status.Errorf(codes.InvalidArgument, "need to specify a gateway")
	}

	err = s.db.AddGateway(ctx, gw.Name, gw.Endpoint, gw.PublicKey)
	if err != nil {
		return nil, status.Errorf(codes.DataLoss, err.Error())
	}

	gw, err = s.db.ReadGateway(ctx, gw.Name)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "gateway has been added, but reading back from database returned error: %s", err.Error())
	}

	return &pb.EnrollGatewayResponse{
		Gateway: gw,
	}, nil
}
