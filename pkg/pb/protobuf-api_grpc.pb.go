// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.20.3
// source: pkg/pb/protobuf-api.proto

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	DeviceHelper_Configure_FullMethodName = "/naisdevice.DeviceHelper/Configure"
	DeviceHelper_Teardown_FullMethodName  = "/naisdevice.DeviceHelper/Teardown"
	DeviceHelper_Upgrade_FullMethodName   = "/naisdevice.DeviceHelper/Upgrade"
	DeviceHelper_GetSerial_FullMethodName = "/naisdevice.DeviceHelper/GetSerial"
)

// DeviceHelperClient is the client API for DeviceHelper service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DeviceHelperClient interface {
	// Push and apply new VPN configuration.
	Configure(ctx context.Context, in *Configuration, opts ...grpc.CallOption) (*ConfigureResponse, error)
	// Delete VPN configuration and shut down connections.
	Teardown(ctx context.Context, in *TeardownRequest, opts ...grpc.CallOption) (*TeardownResponse, error)
	// Install the newest version of naisdevice.
	Upgrade(ctx context.Context, in *UpgradeRequest, opts ...grpc.CallOption) (*UpgradeResponse, error)
	GetSerial(ctx context.Context, in *GetSerialRequest, opts ...grpc.CallOption) (*GetSerialResponse, error)
}

type deviceHelperClient struct {
	cc grpc.ClientConnInterface
}

func NewDeviceHelperClient(cc grpc.ClientConnInterface) DeviceHelperClient {
	return &deviceHelperClient{cc}
}

func (c *deviceHelperClient) Configure(ctx context.Context, in *Configuration, opts ...grpc.CallOption) (*ConfigureResponse, error) {
	out := new(ConfigureResponse)
	err := c.cc.Invoke(ctx, DeviceHelper_Configure_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deviceHelperClient) Teardown(ctx context.Context, in *TeardownRequest, opts ...grpc.CallOption) (*TeardownResponse, error) {
	out := new(TeardownResponse)
	err := c.cc.Invoke(ctx, DeviceHelper_Teardown_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deviceHelperClient) Upgrade(ctx context.Context, in *UpgradeRequest, opts ...grpc.CallOption) (*UpgradeResponse, error) {
	out := new(UpgradeResponse)
	err := c.cc.Invoke(ctx, DeviceHelper_Upgrade_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deviceHelperClient) GetSerial(ctx context.Context, in *GetSerialRequest, opts ...grpc.CallOption) (*GetSerialResponse, error) {
	out := new(GetSerialResponse)
	err := c.cc.Invoke(ctx, DeviceHelper_GetSerial_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeviceHelperServer is the server API for DeviceHelper service.
// All implementations must embed UnimplementedDeviceHelperServer
// for forward compatibility
type DeviceHelperServer interface {
	// Push and apply new VPN configuration.
	Configure(context.Context, *Configuration) (*ConfigureResponse, error)
	// Delete VPN configuration and shut down connections.
	Teardown(context.Context, *TeardownRequest) (*TeardownResponse, error)
	// Install the newest version of naisdevice.
	Upgrade(context.Context, *UpgradeRequest) (*UpgradeResponse, error)
	GetSerial(context.Context, *GetSerialRequest) (*GetSerialResponse, error)
	mustEmbedUnimplementedDeviceHelperServer()
}

// UnimplementedDeviceHelperServer must be embedded to have forward compatible implementations.
type UnimplementedDeviceHelperServer struct {
}

func (UnimplementedDeviceHelperServer) Configure(context.Context, *Configuration) (*ConfigureResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Configure not implemented")
}
func (UnimplementedDeviceHelperServer) Teardown(context.Context, *TeardownRequest) (*TeardownResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Teardown not implemented")
}
func (UnimplementedDeviceHelperServer) Upgrade(context.Context, *UpgradeRequest) (*UpgradeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Upgrade not implemented")
}
func (UnimplementedDeviceHelperServer) GetSerial(context.Context, *GetSerialRequest) (*GetSerialResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSerial not implemented")
}
func (UnimplementedDeviceHelperServer) mustEmbedUnimplementedDeviceHelperServer() {}

// UnsafeDeviceHelperServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DeviceHelperServer will
// result in compilation errors.
type UnsafeDeviceHelperServer interface {
	mustEmbedUnimplementedDeviceHelperServer()
}

func RegisterDeviceHelperServer(s grpc.ServiceRegistrar, srv DeviceHelperServer) {
	s.RegisterService(&DeviceHelper_ServiceDesc, srv)
}

func _DeviceHelper_Configure_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Configuration)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeviceHelperServer).Configure(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeviceHelper_Configure_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeviceHelperServer).Configure(ctx, req.(*Configuration))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeviceHelper_Teardown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TeardownRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeviceHelperServer).Teardown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeviceHelper_Teardown_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeviceHelperServer).Teardown(ctx, req.(*TeardownRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeviceHelper_Upgrade_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpgradeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeviceHelperServer).Upgrade(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeviceHelper_Upgrade_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeviceHelperServer).Upgrade(ctx, req.(*UpgradeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeviceHelper_GetSerial_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetSerialRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeviceHelperServer).GetSerial(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeviceHelper_GetSerial_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeviceHelperServer).GetSerial(ctx, req.(*GetSerialRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DeviceHelper_ServiceDesc is the grpc.ServiceDesc for DeviceHelper service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DeviceHelper_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "naisdevice.DeviceHelper",
	HandlerType: (*DeviceHelperServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Configure",
			Handler:    _DeviceHelper_Configure_Handler,
		},
		{
			MethodName: "Teardown",
			Handler:    _DeviceHelper_Teardown_Handler,
		},
		{
			MethodName: "Upgrade",
			Handler:    _DeviceHelper_Upgrade_Handler,
		},
		{
			MethodName: "GetSerial",
			Handler:    _DeviceHelper_GetSerial_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pkg/pb/protobuf-api.proto",
}

const (
	DeviceAgent_Status_FullMethodName                = "/naisdevice.DeviceAgent/Status"
	DeviceAgent_ConfigureJITA_FullMethodName         = "/naisdevice.DeviceAgent/ConfigureJITA"
	DeviceAgent_Login_FullMethodName                 = "/naisdevice.DeviceAgent/Login"
	DeviceAgent_Logout_FullMethodName                = "/naisdevice.DeviceAgent/Logout"
	DeviceAgent_SetActiveTenant_FullMethodName       = "/naisdevice.DeviceAgent/SetActiveTenant"
	DeviceAgent_SetAgentConfiguration_FullMethodName = "/naisdevice.DeviceAgent/SetAgentConfiguration"
	DeviceAgent_GetAgentConfiguration_FullMethodName = "/naisdevice.DeviceAgent/GetAgentConfiguration"
)

// DeviceAgentClient is the client API for DeviceAgent service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DeviceAgentClient interface {
	// DeviceAgent will stream all state changes on this endpoint.
	// Use Status() to continuously monitor the current Agent status.
	Status(ctx context.Context, in *AgentStatusRequest, opts ...grpc.CallOption) (DeviceAgent_StatusClient, error)
	// Open the JITA form in a web browser.
	ConfigureJITA(ctx context.Context, in *ConfigureJITARequest, opts ...grpc.CallOption) (*ConfigureJITAResponse, error)
	// Log in to API server, enabling access to protected resources.
	Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error)
	// Log out of API server, shutting down all VPN connections.
	Logout(ctx context.Context, in *LogoutRequest, opts ...grpc.CallOption) (*LogoutResponse, error)
	// Set active tenant
	SetActiveTenant(ctx context.Context, in *SetActiveTenantRequest, opts ...grpc.CallOption) (*SetActiveTenantResponse, error)
	// Set device agent configuration
	SetAgentConfiguration(ctx context.Context, in *SetAgentConfigurationRequest, opts ...grpc.CallOption) (*SetAgentConfigurationResponse, error)
	// Get the current configuration for the device agent
	GetAgentConfiguration(ctx context.Context, in *GetAgentConfigurationRequest, opts ...grpc.CallOption) (*GetAgentConfigurationResponse, error)
}

type deviceAgentClient struct {
	cc grpc.ClientConnInterface
}

func NewDeviceAgentClient(cc grpc.ClientConnInterface) DeviceAgentClient {
	return &deviceAgentClient{cc}
}

func (c *deviceAgentClient) Status(ctx context.Context, in *AgentStatusRequest, opts ...grpc.CallOption) (DeviceAgent_StatusClient, error) {
	stream, err := c.cc.NewStream(ctx, &DeviceAgent_ServiceDesc.Streams[0], DeviceAgent_Status_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &deviceAgentStatusClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type DeviceAgent_StatusClient interface {
	Recv() (*AgentStatus, error)
	grpc.ClientStream
}

type deviceAgentStatusClient struct {
	grpc.ClientStream
}

func (x *deviceAgentStatusClient) Recv() (*AgentStatus, error) {
	m := new(AgentStatus)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *deviceAgentClient) ConfigureJITA(ctx context.Context, in *ConfigureJITARequest, opts ...grpc.CallOption) (*ConfigureJITAResponse, error) {
	out := new(ConfigureJITAResponse)
	err := c.cc.Invoke(ctx, DeviceAgent_ConfigureJITA_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deviceAgentClient) Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error) {
	out := new(LoginResponse)
	err := c.cc.Invoke(ctx, DeviceAgent_Login_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deviceAgentClient) Logout(ctx context.Context, in *LogoutRequest, opts ...grpc.CallOption) (*LogoutResponse, error) {
	out := new(LogoutResponse)
	err := c.cc.Invoke(ctx, DeviceAgent_Logout_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deviceAgentClient) SetActiveTenant(ctx context.Context, in *SetActiveTenantRequest, opts ...grpc.CallOption) (*SetActiveTenantResponse, error) {
	out := new(SetActiveTenantResponse)
	err := c.cc.Invoke(ctx, DeviceAgent_SetActiveTenant_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deviceAgentClient) SetAgentConfiguration(ctx context.Context, in *SetAgentConfigurationRequest, opts ...grpc.CallOption) (*SetAgentConfigurationResponse, error) {
	out := new(SetAgentConfigurationResponse)
	err := c.cc.Invoke(ctx, DeviceAgent_SetAgentConfiguration_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deviceAgentClient) GetAgentConfiguration(ctx context.Context, in *GetAgentConfigurationRequest, opts ...grpc.CallOption) (*GetAgentConfigurationResponse, error) {
	out := new(GetAgentConfigurationResponse)
	err := c.cc.Invoke(ctx, DeviceAgent_GetAgentConfiguration_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeviceAgentServer is the server API for DeviceAgent service.
// All implementations must embed UnimplementedDeviceAgentServer
// for forward compatibility
type DeviceAgentServer interface {
	// DeviceAgent will stream all state changes on this endpoint.
	// Use Status() to continuously monitor the current Agent status.
	Status(*AgentStatusRequest, DeviceAgent_StatusServer) error
	// Open the JITA form in a web browser.
	ConfigureJITA(context.Context, *ConfigureJITARequest) (*ConfigureJITAResponse, error)
	// Log in to API server, enabling access to protected resources.
	Login(context.Context, *LoginRequest) (*LoginResponse, error)
	// Log out of API server, shutting down all VPN connections.
	Logout(context.Context, *LogoutRequest) (*LogoutResponse, error)
	// Set active tenant
	SetActiveTenant(context.Context, *SetActiveTenantRequest) (*SetActiveTenantResponse, error)
	// Set device agent configuration
	SetAgentConfiguration(context.Context, *SetAgentConfigurationRequest) (*SetAgentConfigurationResponse, error)
	// Get the current configuration for the device agent
	GetAgentConfiguration(context.Context, *GetAgentConfigurationRequest) (*GetAgentConfigurationResponse, error)
	mustEmbedUnimplementedDeviceAgentServer()
}

// UnimplementedDeviceAgentServer must be embedded to have forward compatible implementations.
type UnimplementedDeviceAgentServer struct {
}

func (UnimplementedDeviceAgentServer) Status(*AgentStatusRequest, DeviceAgent_StatusServer) error {
	return status.Errorf(codes.Unimplemented, "method Status not implemented")
}
func (UnimplementedDeviceAgentServer) ConfigureJITA(context.Context, *ConfigureJITARequest) (*ConfigureJITAResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConfigureJITA not implemented")
}
func (UnimplementedDeviceAgentServer) Login(context.Context, *LoginRequest) (*LoginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Login not implemented")
}
func (UnimplementedDeviceAgentServer) Logout(context.Context, *LogoutRequest) (*LogoutResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Logout not implemented")
}
func (UnimplementedDeviceAgentServer) SetActiveTenant(context.Context, *SetActiveTenantRequest) (*SetActiveTenantResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetActiveTenant not implemented")
}
func (UnimplementedDeviceAgentServer) SetAgentConfiguration(context.Context, *SetAgentConfigurationRequest) (*SetAgentConfigurationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetAgentConfiguration not implemented")
}
func (UnimplementedDeviceAgentServer) GetAgentConfiguration(context.Context, *GetAgentConfigurationRequest) (*GetAgentConfigurationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAgentConfiguration not implemented")
}
func (UnimplementedDeviceAgentServer) mustEmbedUnimplementedDeviceAgentServer() {}

// UnsafeDeviceAgentServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DeviceAgentServer will
// result in compilation errors.
type UnsafeDeviceAgentServer interface {
	mustEmbedUnimplementedDeviceAgentServer()
}

func RegisterDeviceAgentServer(s grpc.ServiceRegistrar, srv DeviceAgentServer) {
	s.RegisterService(&DeviceAgent_ServiceDesc, srv)
}

func _DeviceAgent_Status_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(AgentStatusRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(DeviceAgentServer).Status(m, &deviceAgentStatusServer{stream})
}

type DeviceAgent_StatusServer interface {
	Send(*AgentStatus) error
	grpc.ServerStream
}

type deviceAgentStatusServer struct {
	grpc.ServerStream
}

func (x *deviceAgentStatusServer) Send(m *AgentStatus) error {
	return x.ServerStream.SendMsg(m)
}

func _DeviceAgent_ConfigureJITA_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConfigureJITARequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeviceAgentServer).ConfigureJITA(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeviceAgent_ConfigureJITA_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeviceAgentServer).ConfigureJITA(ctx, req.(*ConfigureJITARequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeviceAgent_Login_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LoginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeviceAgentServer).Login(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeviceAgent_Login_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeviceAgentServer).Login(ctx, req.(*LoginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeviceAgent_Logout_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LogoutRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeviceAgentServer).Logout(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeviceAgent_Logout_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeviceAgentServer).Logout(ctx, req.(*LogoutRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeviceAgent_SetActiveTenant_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetActiveTenantRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeviceAgentServer).SetActiveTenant(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeviceAgent_SetActiveTenant_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeviceAgentServer).SetActiveTenant(ctx, req.(*SetActiveTenantRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeviceAgent_SetAgentConfiguration_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetAgentConfigurationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeviceAgentServer).SetAgentConfiguration(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeviceAgent_SetAgentConfiguration_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeviceAgentServer).SetAgentConfiguration(ctx, req.(*SetAgentConfigurationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeviceAgent_GetAgentConfiguration_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetAgentConfigurationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeviceAgentServer).GetAgentConfiguration(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeviceAgent_GetAgentConfiguration_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeviceAgentServer).GetAgentConfiguration(ctx, req.(*GetAgentConfigurationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DeviceAgent_ServiceDesc is the grpc.ServiceDesc for DeviceAgent service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DeviceAgent_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "naisdevice.DeviceAgent",
	HandlerType: (*DeviceAgentServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ConfigureJITA",
			Handler:    _DeviceAgent_ConfigureJITA_Handler,
		},
		{
			MethodName: "Login",
			Handler:    _DeviceAgent_Login_Handler,
		},
		{
			MethodName: "Logout",
			Handler:    _DeviceAgent_Logout_Handler,
		},
		{
			MethodName: "SetActiveTenant",
			Handler:    _DeviceAgent_SetActiveTenant_Handler,
		},
		{
			MethodName: "SetAgentConfiguration",
			Handler:    _DeviceAgent_SetAgentConfiguration_Handler,
		},
		{
			MethodName: "GetAgentConfiguration",
			Handler:    _DeviceAgent_GetAgentConfiguration_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Status",
			Handler:       _DeviceAgent_Status_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "pkg/pb/protobuf-api.proto",
}

const (
	APIServer_Login_FullMethodName                   = "/naisdevice.APIServer/Login"
	APIServer_GetDeviceConfiguration_FullMethodName  = "/naisdevice.APIServer/GetDeviceConfiguration"
	APIServer_GetGatewayConfiguration_FullMethodName = "/naisdevice.APIServer/GetGatewayConfiguration"
	APIServer_GetGateway_FullMethodName              = "/naisdevice.APIServer/GetGateway"
	APIServer_ListGateways_FullMethodName            = "/naisdevice.APIServer/ListGateways"
	APIServer_EnrollGateway_FullMethodName           = "/naisdevice.APIServer/EnrollGateway"
	APIServer_UpdateGateway_FullMethodName           = "/naisdevice.APIServer/UpdateGateway"
)

// APIServerClient is the client API for APIServer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type APIServerClient interface {
	// Exchange an access token for a session
	Login(ctx context.Context, in *APIServerLoginRequest, opts ...grpc.CallOption) (*APIServerLoginResponse, error)
	// Set up a client->server request for continuous streaming of new configuration
	GetDeviceConfiguration(ctx context.Context, in *GetDeviceConfigurationRequest, opts ...grpc.CallOption) (APIServer_GetDeviceConfigurationClient, error)
	// Set up continuous streaming of new gateway configuration
	GetGatewayConfiguration(ctx context.Context, in *GetGatewayConfigurationRequest, opts ...grpc.CallOption) (APIServer_GetGatewayConfigurationClient, error)
	// Admin endpoint for retrieving a single gateway
	GetGateway(ctx context.Context, in *ModifyGatewayRequest, opts ...grpc.CallOption) (*Gateway, error)
	// Admin endpoint for listing out gateways registered in database
	ListGateways(ctx context.Context, in *ListGatewayRequest, opts ...grpc.CallOption) (APIServer_ListGatewaysClient, error)
	// Admin endpoint for adding gateway credentials to the database
	EnrollGateway(ctx context.Context, in *ModifyGatewayRequest, opts ...grpc.CallOption) (*ModifyGatewayResponse, error)
	// Admin endpoint for adding gateway credentials to the database
	UpdateGateway(ctx context.Context, in *ModifyGatewayRequest, opts ...grpc.CallOption) (*ModifyGatewayResponse, error)
}

type aPIServerClient struct {
	cc grpc.ClientConnInterface
}

func NewAPIServerClient(cc grpc.ClientConnInterface) APIServerClient {
	return &aPIServerClient{cc}
}

func (c *aPIServerClient) Login(ctx context.Context, in *APIServerLoginRequest, opts ...grpc.CallOption) (*APIServerLoginResponse, error) {
	out := new(APIServerLoginResponse)
	err := c.cc.Invoke(ctx, APIServer_Login_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *aPIServerClient) GetDeviceConfiguration(ctx context.Context, in *GetDeviceConfigurationRequest, opts ...grpc.CallOption) (APIServer_GetDeviceConfigurationClient, error) {
	stream, err := c.cc.NewStream(ctx, &APIServer_ServiceDesc.Streams[0], APIServer_GetDeviceConfiguration_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &aPIServerGetDeviceConfigurationClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type APIServer_GetDeviceConfigurationClient interface {
	Recv() (*GetDeviceConfigurationResponse, error)
	grpc.ClientStream
}

type aPIServerGetDeviceConfigurationClient struct {
	grpc.ClientStream
}

func (x *aPIServerGetDeviceConfigurationClient) Recv() (*GetDeviceConfigurationResponse, error) {
	m := new(GetDeviceConfigurationResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *aPIServerClient) GetGatewayConfiguration(ctx context.Context, in *GetGatewayConfigurationRequest, opts ...grpc.CallOption) (APIServer_GetGatewayConfigurationClient, error) {
	stream, err := c.cc.NewStream(ctx, &APIServer_ServiceDesc.Streams[1], APIServer_GetGatewayConfiguration_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &aPIServerGetGatewayConfigurationClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type APIServer_GetGatewayConfigurationClient interface {
	Recv() (*GetGatewayConfigurationResponse, error)
	grpc.ClientStream
}

type aPIServerGetGatewayConfigurationClient struct {
	grpc.ClientStream
}

func (x *aPIServerGetGatewayConfigurationClient) Recv() (*GetGatewayConfigurationResponse, error) {
	m := new(GetGatewayConfigurationResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *aPIServerClient) GetGateway(ctx context.Context, in *ModifyGatewayRequest, opts ...grpc.CallOption) (*Gateway, error) {
	out := new(Gateway)
	err := c.cc.Invoke(ctx, APIServer_GetGateway_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *aPIServerClient) ListGateways(ctx context.Context, in *ListGatewayRequest, opts ...grpc.CallOption) (APIServer_ListGatewaysClient, error) {
	stream, err := c.cc.NewStream(ctx, &APIServer_ServiceDesc.Streams[2], APIServer_ListGateways_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &aPIServerListGatewaysClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type APIServer_ListGatewaysClient interface {
	Recv() (*Gateway, error)
	grpc.ClientStream
}

type aPIServerListGatewaysClient struct {
	grpc.ClientStream
}

func (x *aPIServerListGatewaysClient) Recv() (*Gateway, error) {
	m := new(Gateway)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *aPIServerClient) EnrollGateway(ctx context.Context, in *ModifyGatewayRequest, opts ...grpc.CallOption) (*ModifyGatewayResponse, error) {
	out := new(ModifyGatewayResponse)
	err := c.cc.Invoke(ctx, APIServer_EnrollGateway_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *aPIServerClient) UpdateGateway(ctx context.Context, in *ModifyGatewayRequest, opts ...grpc.CallOption) (*ModifyGatewayResponse, error) {
	out := new(ModifyGatewayResponse)
	err := c.cc.Invoke(ctx, APIServer_UpdateGateway_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// APIServerServer is the server API for APIServer service.
// All implementations must embed UnimplementedAPIServerServer
// for forward compatibility
type APIServerServer interface {
	// Exchange an access token for a session
	Login(context.Context, *APIServerLoginRequest) (*APIServerLoginResponse, error)
	// Set up a client->server request for continuous streaming of new configuration
	GetDeviceConfiguration(*GetDeviceConfigurationRequest, APIServer_GetDeviceConfigurationServer) error
	// Set up continuous streaming of new gateway configuration
	GetGatewayConfiguration(*GetGatewayConfigurationRequest, APIServer_GetGatewayConfigurationServer) error
	// Admin endpoint for retrieving a single gateway
	GetGateway(context.Context, *ModifyGatewayRequest) (*Gateway, error)
	// Admin endpoint for listing out gateways registered in database
	ListGateways(*ListGatewayRequest, APIServer_ListGatewaysServer) error
	// Admin endpoint for adding gateway credentials to the database
	EnrollGateway(context.Context, *ModifyGatewayRequest) (*ModifyGatewayResponse, error)
	// Admin endpoint for adding gateway credentials to the database
	UpdateGateway(context.Context, *ModifyGatewayRequest) (*ModifyGatewayResponse, error)
	mustEmbedUnimplementedAPIServerServer()
}

// UnimplementedAPIServerServer must be embedded to have forward compatible implementations.
type UnimplementedAPIServerServer struct {
}

func (UnimplementedAPIServerServer) Login(context.Context, *APIServerLoginRequest) (*APIServerLoginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Login not implemented")
}
func (UnimplementedAPIServerServer) GetDeviceConfiguration(*GetDeviceConfigurationRequest, APIServer_GetDeviceConfigurationServer) error {
	return status.Errorf(codes.Unimplemented, "method GetDeviceConfiguration not implemented")
}
func (UnimplementedAPIServerServer) GetGatewayConfiguration(*GetGatewayConfigurationRequest, APIServer_GetGatewayConfigurationServer) error {
	return status.Errorf(codes.Unimplemented, "method GetGatewayConfiguration not implemented")
}
func (UnimplementedAPIServerServer) GetGateway(context.Context, *ModifyGatewayRequest) (*Gateway, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetGateway not implemented")
}
func (UnimplementedAPIServerServer) ListGateways(*ListGatewayRequest, APIServer_ListGatewaysServer) error {
	return status.Errorf(codes.Unimplemented, "method ListGateways not implemented")
}
func (UnimplementedAPIServerServer) EnrollGateway(context.Context, *ModifyGatewayRequest) (*ModifyGatewayResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EnrollGateway not implemented")
}
func (UnimplementedAPIServerServer) UpdateGateway(context.Context, *ModifyGatewayRequest) (*ModifyGatewayResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateGateway not implemented")
}
func (UnimplementedAPIServerServer) mustEmbedUnimplementedAPIServerServer() {}

// UnsafeAPIServerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to APIServerServer will
// result in compilation errors.
type UnsafeAPIServerServer interface {
	mustEmbedUnimplementedAPIServerServer()
}

func RegisterAPIServerServer(s grpc.ServiceRegistrar, srv APIServerServer) {
	s.RegisterService(&APIServer_ServiceDesc, srv)
}

func _APIServer_Login_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(APIServerLoginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(APIServerServer).Login(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: APIServer_Login_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(APIServerServer).Login(ctx, req.(*APIServerLoginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _APIServer_GetDeviceConfiguration_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetDeviceConfigurationRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(APIServerServer).GetDeviceConfiguration(m, &aPIServerGetDeviceConfigurationServer{stream})
}

type APIServer_GetDeviceConfigurationServer interface {
	Send(*GetDeviceConfigurationResponse) error
	grpc.ServerStream
}

type aPIServerGetDeviceConfigurationServer struct {
	grpc.ServerStream
}

func (x *aPIServerGetDeviceConfigurationServer) Send(m *GetDeviceConfigurationResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _APIServer_GetGatewayConfiguration_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetGatewayConfigurationRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(APIServerServer).GetGatewayConfiguration(m, &aPIServerGetGatewayConfigurationServer{stream})
}

type APIServer_GetGatewayConfigurationServer interface {
	Send(*GetGatewayConfigurationResponse) error
	grpc.ServerStream
}

type aPIServerGetGatewayConfigurationServer struct {
	grpc.ServerStream
}

func (x *aPIServerGetGatewayConfigurationServer) Send(m *GetGatewayConfigurationResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _APIServer_GetGateway_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ModifyGatewayRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(APIServerServer).GetGateway(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: APIServer_GetGateway_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(APIServerServer).GetGateway(ctx, req.(*ModifyGatewayRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _APIServer_ListGateways_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ListGatewayRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(APIServerServer).ListGateways(m, &aPIServerListGatewaysServer{stream})
}

type APIServer_ListGatewaysServer interface {
	Send(*Gateway) error
	grpc.ServerStream
}

type aPIServerListGatewaysServer struct {
	grpc.ServerStream
}

func (x *aPIServerListGatewaysServer) Send(m *Gateway) error {
	return x.ServerStream.SendMsg(m)
}

func _APIServer_EnrollGateway_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ModifyGatewayRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(APIServerServer).EnrollGateway(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: APIServer_EnrollGateway_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(APIServerServer).EnrollGateway(ctx, req.(*ModifyGatewayRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _APIServer_UpdateGateway_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ModifyGatewayRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(APIServerServer).UpdateGateway(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: APIServer_UpdateGateway_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(APIServerServer).UpdateGateway(ctx, req.(*ModifyGatewayRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// APIServer_ServiceDesc is the grpc.ServiceDesc for APIServer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var APIServer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "naisdevice.APIServer",
	HandlerType: (*APIServerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Login",
			Handler:    _APIServer_Login_Handler,
		},
		{
			MethodName: "GetGateway",
			Handler:    _APIServer_GetGateway_Handler,
		},
		{
			MethodName: "EnrollGateway",
			Handler:    _APIServer_EnrollGateway_Handler,
		},
		{
			MethodName: "UpdateGateway",
			Handler:    _APIServer_UpdateGateway_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetDeviceConfiguration",
			Handler:       _APIServer_GetDeviceConfiguration_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetGatewayConfiguration",
			Handler:       _APIServer_GetGatewayConfiguration_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "ListGateways",
			Handler:       _APIServer_ListGateways_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "pkg/pb/protobuf-api.proto",
}
