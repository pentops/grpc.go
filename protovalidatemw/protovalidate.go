package protovalidatemw

import (
	"context"
	"fmt"
	"strings"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	protovalidate "github.com/bufbuild/protovalidate-go"
	spb "google.golang.org/genproto/googleapis/rpc/status"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type config struct {
	reply     bool
	validator *protovalidate.Validator
}

type Option func(*config)

func WithReply() Option {
	return func(c *config) {
		c.reply = true
	}
}

func WithValidator(v *protovalidate.Validator) Option {
	return func(c *config) {
		c.validator = v
	}
}

func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.validator == nil {
		validator, err := protovalidate.New()
		if err != nil {
			panic(err)
		}
		cfg.validator = validator
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := runValidate(cfg.validator, req, "request", codes.InvalidArgument); err != nil {
			return nil, err
		}

		res, err := handler(ctx, req)
		if err != nil {
			return nil, err
		}

		if cfg.reply {
			if err := runValidate(cfg.validator, res, "reply", codes.Internal); err != nil {
				return nil, err
			}
		}

		return res, nil
	}
}

func runValidate(v *protovalidate.Validator, msg any, part string, code codes.Code) error {

	protoMsg, ok := msg.(proto.Message)
	if !ok {
		return fmt.Errorf("not a proto message")
	}

	err := v.Validate(protoMsg)
	if err == nil {
		return nil
	}

	isValidateError, ok := err.(*protovalidate.ValidationError)
	if !ok {
		return fmt.Errorf("unknown error type %T", err)
	}

	errorAny, err := anypb.New(isValidateError.ToProto())
	if err != nil {
		return err
	}

	first := isValidateError.Violations[0]
	fieldPath := make([]string, 0, len(first.Proto.Field.Elements))
	for _, field := range first.Proto.Field.Elements {
		fieldName := field.GetFieldName()
		switch st := field.Subscript.(type) {
		case *validate.FieldPathElement_Index:
			fieldName += fmt.Sprintf("[%d]", st.Index)
		case *validate.FieldPathElement_BoolKey:
			fieldName += fmt.Sprintf("[%t]", st.BoolKey)
		case *validate.FieldPathElement_StringKey:
			fieldName += fmt.Sprintf("[%s]", st.StringKey)
		case *validate.FieldPathElement_IntKey:
			fieldName += fmt.Sprintf("[%d]", st.IntKey)
		case *validate.FieldPathElement_UintKey:
			fieldName += fmt.Sprintf("[%d]", st.UintKey)
		}
		fieldPath = append(fieldPath, fieldName)
	}

	statusError := spb.Status{
		Code:    int32(code),
		Message: fmt.Sprintf("invalid %s: %s: %s", part, strings.Join(fieldPath, "."), stringVal(first.Proto.Message)),
		Details: []*anypb.Any{errorAny},
	}
	return status.FromProto(&statusError).Err()
}

func stringVal(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
