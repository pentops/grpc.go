package versionmw

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryServerInterceptor(version string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		header := metadata.Pairs("X-Version", version)
		if err := grpc.SetHeader(ctx, header); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}
