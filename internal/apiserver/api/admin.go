package api

import (
	"context"
	"fmt"

	"github.com/nais/device/pkg/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *grpcServer) addOrUpdateGateway(ctx context.Context, r *pb.ModifyGatewayRequest, callback func(context.Context, *pb.Gateway) error) (*pb.ModifyGatewayResponse, error) {
	err := s.adminAuth.Authenticate(ctx, r.GetUsername(), r.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authenticate: %v", err)
	}

	gw := r.GetGateway()
	if gw == nil {
		return nil, status.Errorf(codes.InvalidArgument, "get gateway: %v", err)
	}

	err = callback(ctx, gw)
	if err != nil {
		return nil, status.Errorf(codes.DataLoss, "callback: %v", err)
	}

	gw, err = s.db.ReadGateway(ctx, gw.Name)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "gateway has been added, but reading back from database returned error: %v", err)
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
	err := s.adminAuth.Authenticate(ctx, r.GetUsername(), r.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authenticate: %v", err)
	}

	return s.db.ReadGateway(ctx, r.GetGateway().GetName())
}

func (s *grpcServer) ListGateways(request *pb.ListGatewayRequest, stream pb.APIServer_ListGatewaysServer) error {
	err := authenticateAny(stream.Context(), request.GetUsername(), request.GetPassword(), s.adminAuth, s.prometheusAuth)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	gateways, err := s.db.ReadGateways(stream.Context())
	if err != nil {
		return status.Error(codes.Unavailable, err.Error())
	}
	for _, gw := range gateways {
		err = stream.Send(gw)
		if err != nil {
			return status.Error(codes.Aborted, err.Error())
		}
	}

	return nil
}

func (s *grpcServer) GetSessions(ctx context.Context, r *pb.GetSessionsRequest) (*pb.GetSessionsResponse, error) {
	err := s.adminAuth.Authenticate(ctx, r.GetUsername(), r.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authenticate: %v", err)
	}

	return &pb.GetSessionsResponse{
		Sessions: s.sessionStore.All(),
	}, nil
}

func (s *grpcServer) GetKolideCache(ctx context.Context, r *pb.GetKolideCacheRequest) (*pb.GetKolideCacheResponse, error) {
	err := s.adminAuth.Authenticate(ctx, r.GetUsername(), r.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authenticate: %v", err)
	}

	if s.kolideClient == nil {
		return nil, fmt.Errorf("kolide client not configured")
	}

	return &pb.GetKolideCacheResponse{}, nil
}
