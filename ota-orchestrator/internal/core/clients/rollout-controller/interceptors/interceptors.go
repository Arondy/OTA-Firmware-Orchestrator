package interceptors

import (
	"context"

	"connectrpc.com/connect"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
)

func NewInterceptorsOption() connect.ClientOption {
	return connect.WithInterceptors(
		RequestID(),
	)
}

func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(config.CtxKeyRequestID{}).(string); ok {
		return id
	}
	return ""
}

func RequestID() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if id := RequestIDFromContext(ctx); id != "" {
				req.Header().Set(config.RequestIDHeader, id)
			}
			return next(ctx, req)
		}
	})
}
