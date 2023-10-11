package integrationtest_test

func NewHelper(t *testing.T, ctx context.Context) *grpc.Server {
	sessions := auth.NewMockSessionStore(t)
	deviceAuth := auth.NewMockAuthenticator(sessions)
	gatewayAuth := auth.NewMockAPIKeyAuthenticator()

	j := jita.New("user", "pass", "url")

	impl := api.NewGRPCServer(ctx, db, deviceAuth, nil, gatewayAuth, nil, j, sessions)
	server := grpc.NewServer()
	pb.RegisterAPIServerServer(server, impl)

	return server

}
