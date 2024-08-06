package protovalidatemw

import (
	"context"
	"fmt"

	protovalidate "github.com/bufbuild/protovalidate-go"
	spb "google.golang.org/genproto/googleapis/rpc/status"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	validator, err := protovalidate.New()
	if err != nil {
		panic(err)
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		protoReq, ok := req.(proto.Message)
		if !ok {
			return nil, fmt.Errorf("request is not a proto message")
		}

		if err := validator.Validate(protoReq); err != nil {

			isValidateError, ok := err.(*protovalidate.ValidationError)
			if !ok {
				return nil, fmt.Errorf("unknown error type %T", err)
			}

			errorAny, err := anypb.New(isValidateError.ToProto())
			if err != nil {
				return nil, err
			}

			first := isValidateError.Violations[0]

			statusError := spb.Status{
				Code:    int32(codes.InvalidArgument),
				Message: fmt.Sprintf("invalid request: %s: %s", first.FieldPath, first.Message),
				Details: []*anypb.Any{errorAny},
			}
			return nil, status.FromProto(&statusError).Err()
		}

		return handler(ctx, req)
	}
}
