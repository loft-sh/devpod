package network

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/mwitkow/grpc-proxy/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"tailscale.com/tsnet"
)

// GrpcDirector handles the logic for directing gRPC requests.
type GrpcDirector struct {
	log      log.Logger
	tsServer *tsnet.Server
}

// NewGrpcDirector creates a new GrpcDirector.
func NewGrpcDirector(tsSrv *tsnet.Server, logger log.Logger) *GrpcDirector {
	return &GrpcDirector{
		log:      logger,
		tsServer: tsSrv,
	}
}

func (d *GrpcDirector) tsDialer(ctx context.Context, addr string) (net.Conn, error) {
	d.log.Debugf("GrpcDirector: Dialing target %s via tsnet", addr)
	conn, err := d.tsServer.Dial(ctx, "tcp", addr)
	if err != nil {
		d.log.Errorf("GrpcDirector: Failed to dial target %s via tsnet: %v", addr, err)
		return nil, fmt.Errorf("tsnet dial failed for %s: %w", addr, err)
	}
	d.log.Debugf("GrpcDirector: Successfully dialed %s via tsnet", addr)
	return conn, nil
}

func (d *GrpcDirector) DirectorFunc(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		d.log.Warnf("NetworkProxyService: gRPC: Director missing incoming metadata for call %q", fullMethodName)
		return nil, nil, status.Errorf(codes.InvalidArgument, "missing metadata")
	}
	mdCopy := md.Copy()

	targetHosts := mdCopy.Get(HeaderTargetHost)
	targetPorts := mdCopy.Get(HeaderTargetPort)
	proxyPorts := mdCopy.Get(HeaderProxyPort)

	if len(targetHosts) == 0 || len(targetPorts) == 0 || len(proxyPorts) == 0 {
		d.log.Errorf("NetworkProxyService: gRPC: Director missing x-target-host, x-proxy-port or x-target-port metadata for call %q", fullMethodName)
		return nil, nil, status.Errorf(codes.InvalidArgument, "missing x-target-host, x-proxy-port or x-target-port metadata")
	}

	proxyPort, err := strconv.Atoi(proxyPorts[0])
	if err != nil {
		d.log.Errorf("NetworkProxyService: gRPC: Invalid x-proxy-port %q: %v", proxyPorts[0], err)
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid x-proxy-port: %v", err)
	}
	// The address we dial via tsnet is the intermediate proxy host and its port
	targetAddr := ts.EnsureURL(targetHosts[0], proxyPort)
	d.log.Debugf("NetworkProxyService: gRPC: Proxying call %q to target %s", fullMethodName, targetAddr)

	conn, err := grpc.DialContext(ctx, targetAddr,
		grpc.WithContextDialer(d.tsDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithCodec(proxy.Codec()),
	)
	if err != nil {
		d.log.Errorf("NetworkProxyService: gRPC: Failed to dial backend %s: %v", targetAddr, err)
		return nil, nil, status.Errorf(codes.Internal, "failed to dial backend: %v", err)
	}

	outCtx := metadata.NewOutgoingContext(ctx, mdCopy)

	return outCtx, conn, nil
}
