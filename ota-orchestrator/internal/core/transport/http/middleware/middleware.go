package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	core_http "github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Middleware func(next http.Handler) http.Handler

func WrapInMiddleware(router http.Handler, logger *zap.SugaredLogger) http.Handler {
	router = Recover(router)
	router = Trace(router)
	router = Logger(logger)(router)
	router = RequestID(router)
	return router
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger := config.LoggerFromContext(r.Context())
				logger.Errorw("unexpected panic", "error", err)
				handlers.WriteInternalServerError(w, logger)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := r.Header.Get(config.RequestIDHeader)
		if idStr == "" {
			id, err := uuid.NewV7()
			if err != nil {
				id = uuid.Nil
			}

			idStr = id.String()
			r.Header.Set(config.RequestIDHeader, idStr)
		}

		ctx := context.WithValue(r.Context(), config.CtxKeyRequestID{}, idStr)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func Logger(baseLogger *zap.SugaredLogger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := baseLogger.With(
				"method", r.Method,
				"path", r.URL.Path,
				"request_id", r.Header.Get(config.RequestIDHeader),
			)
			ctx := config.LoggerToContext(r.Context(), logger)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := config.LoggerFromContext(r.Context())

		logger.Debug("incoming request")
		rw := core_http.NewResponseWriter(w)
		s := time.Now()
		next.ServeHTTP(rw, r)
		logger.Debugw("sent response", "status_code", rw.GetStatusCode(), "latency", fmt.Sprintf("%.3fs", time.Since(s).Seconds()))
	})
}
