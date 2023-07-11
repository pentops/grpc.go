package grpcerror

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type validatorOne interface {
	Validate() error
}

type validatorAll interface {
	ValidateAll() error
}

func FormatValidationError(err error) error {
	return status.Error(codes.InvalidArgument, err.Error())
}

type Logger interface {
	// This is designed to be compatible with the upcoming slog standard library, the function will be always called
	// as (msg string, 'error' string, the actual error string)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

func UnaryServerInterceptor(logger Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if v, ok := req.(validatorAll); ok {
			if err := v.ValidateAll(); err != nil {
				return nil, FormatValidationError(err)
			}
		} else if v, ok := req.(validatorOne); ok {
			if err := v.Validate(); err != nil {
				return nil, FormatValidationError(err)
			}
		}

		res, err := handler(ctx, req)
		if err != nil {
			if _, ok := status.FromError(err); !ok {
				if logger != nil {
					logger.ErrorContext(ctx, "hiding unhandled error", "error", err.Error())
				}
				return nil, status.Error(codes.Internal, "Internal Error")
			}
			return nil, err
		}
		return res, nil
	}
}
