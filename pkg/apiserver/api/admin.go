package api

import (
	"context"

	"github.com/nais/device/pkg/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *grpcServer) addOrUpdateGateway(ctx context.Context, r *pb.ModifyGatewayRequest, callback func(context.Context, *pb.Gateway) error) (*pb.ModifyGatewayResponse, error) {
	err := s.apikeyAuthenticator.Authenticate(AdminUsername, r.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	}

	gw := r.GetGateway()
	if gw == nil {
		return nil, status.Errorf(codes.InvalidArgument, "need to specify a gateway")
	}

	err = callback(ctx, gw)
	if err != nil {
		return nil, status.Errorf(codes.DataLoss, err.Error())
	}

	gw, err = s.db.ReadGateway(ctx, gw.Name)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "gateway has been added, but reading back from database returned error: %s", err.Error())
	}

	return &pb.ModifyGatewayResponse{
		Gateway: gw,
	}, nil
}

func (s *grpcServer) EnrollGateway(ctx context.Context, r *pb.ModifyGatewayRequest) (*pb.ModifyGatewayResponse, error) {
	return s.addOrUpdateGateway(ctx, r, s.db.AddGateway)
}

func (s *grpcServer) UpdateGateway(ctx context.Context, r *pb.ModifyGatewayRequest) (*pb.ModifyGatewayResponse, error) {
	return s.addOrUpdateGateway(ctx, r, s.db.UpdateGateway)
}

func (s *grpcServer) GetGateway(ctx context.Context, r *pb.ModifyGatewayRequest) (*pb.Gateway, error) {
	err := s.apikeyAuthenticator.Authenticate(AdminUsername, r.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	}

	return s.db.ReadGateway(ctx, r.GetGateway().GetName())
}
