package interceptors

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/config"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const requestIDHeader = "x-request-id"

func NewInterceptorsOption(logger *zap.SugaredLogger) connect.HandlerOption {
	return connect.WithInterceptors(
		RequestID(),
		Logger(logger),
		Trace(),
		Recover(),
	)
}

func Recover() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			defer func() {
				if recErr := recover(); recErr != nil {
					logger := config.LoggerFromContext(ctx)
					logger.Errorw("unexpected panic", "error", recErr)
					err = connect.NewError(connect.CodeInternal, errors.New("internal server error"))
				}
			}()

			return next(ctx, req)
		}
	})
}

func RequestID() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			requestID := req.Header().Get(requestIDHeader)
			if requestID == "" {
				requestID = uuid.NewString()
				req.Header().Set(requestIDHeader, requestID)
			}

			resp, err := next(ctx, req)

			if err != nil {
				if connErr, ok := errors.AsType[*connect.Error](err); ok {
					connErr.Meta().Set(requestIDHeader, requestID)
				} else {
					connErr = connect.NewError(connect.CodeUnknown, err)
					connErr.Meta().Set(requestIDHeader, requestID)
					err = connErr
				}
			}

			if resp != nil {
				resp.Header().Set(requestIDHeader, requestID)
			}

			return resp, err
		}
	})
}

func Logger(baseLogger *zap.SugaredLogger) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			logger := baseLogger.With(
				"procedure", req.Spec().Procedure,
				"request_id", req.Header().Get(requestIDHeader),
			)

			ctx = config.LoggerToContext(ctx, logger)
			return next(ctx, req)
		}
	})
}

func Trace() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			logger := config.LoggerFromContext(ctx)

			logger.Debug("incoming request")
			code := "ok"
			s := time.Now()

			resp, err := next(ctx, req)
			if err != nil {
				code = connect.CodeOf(err).String()
			}

			logger.Debugw("sent response", "code", code, "latency", fmt.Sprintf("%.3fs", time.Since(s).Seconds()))
			return resp, err
		}
	})
}
