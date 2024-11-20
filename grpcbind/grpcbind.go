package grpcbind

import (
	"context"
	"fmt"
	"net"
	"strings"
)

func urlToListener(bind string) (string, string, error) {
	if len(bind) == 0 {
		return "", "", fmt.Errorf("empty bind address")
	}
	if bind[0] == '/' {
		return "unix", bind, nil
	}
	if bind[0] == ':' {
		return "tcp", bind, nil
	}

	parts := strings.SplitN(bind, "://", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unsupported bind address %q, missing scheme", bind)
	}

	switch parts[0] {
	case "unix":
		return "unix", parts[1], nil
	case "tcp":
		return "tcp", parts[1], nil
	default:
		return "", "", fmt.Errorf("unsupported scheme %q", parts[0])
	}

}

// Listen starts a net listener on the given bind address, which may be a TCP
// port or address, or a Unix domain socket.
func Listen(ctx context.Context, bind string) (net.Listener, error) {
	network, addr, err := urlToListener(bind)
	if err != nil {
		return nil, err
	}
	var lc net.ListenConfig
	return lc.Listen(ctx, network, addr)
}

type GRPCServer interface {
	Serve(net.Listener) error
	GracefulStop()
}

// Serve starts the gRPC server on the given listener until the context is done
// or an error is returned.
func Serve(ctx context.Context, server GRPCServer, listener net.Listener) error {
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()
	return server.Serve(listener)
}

// ListenAndServe starts a listener on the given bind address and serves until
// the context is done or an error is returned.
func ListenAndServe(ctx context.Context, server GRPCServer, bind string) error {
	lis, err := Listen(ctx, bind)
	if err != nil {
		return err
	}
	return Serve(ctx, server, lis)
}
