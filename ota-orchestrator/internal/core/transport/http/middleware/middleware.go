package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
	core_http "github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/transport/http"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/transport/http/handlers"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const requestIDHeader = "x-request-id"

type Middleware func(next http.Handler) http.Handler

func WrapInMiddleware(router http.Handler, logger *zap.SugaredLogger) http.Handler {
	router = Trace(router)
	router = Logger(logger)(router)
	router = RequestID(router)
	router = Recover(router)
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
		if r.Header.Get(requestIDHeader) == "" {
			r.Header.Set(requestIDHeader, uuid.NewString())
		}
		next.ServeHTTP(w, r)
	})
}

func Logger(baseLogger *zap.SugaredLogger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := baseLogger.With(
				"method", r.Method,
				"path", r.URL.Path,
				"request_id", r.Header.Get(requestIDHeader),
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
