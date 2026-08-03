package core_http

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
	"go.uber.org/zap"
)

type zapLoggerWriter struct {
	logger *zap.SugaredLogger
}

func (w *zapLoggerWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	w.logger.Errorw("HTTP server internal error", "source_err", msg)
	return len(p), nil
}

type Server struct {
	server *http.Server
	config *config.HTTPServerConfig
	logger *zap.SugaredLogger
}

func NewServer(router http.Handler, config *config.HTTPServerConfig, logger *zap.SugaredLogger) *Server {
	zapWriter := &zapLoggerWriter{logger: logger.Named("ServerErrorLog")}
	stdLogger := log.New(zapWriter, "", 0)

	server := &http.Server{
		Addr:              net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))),
		Handler:           router,
		ErrorLog:          stdLogger,
		ReadHeaderTimeout: 1 * time.Minute,
		IdleTimeout:       1 * time.Minute,
	}

	return &Server{
		server: server,
		config: config,
		logger: logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	ch := make(chan error, 1)

	go func() {
		defer close(ch)
		s.logger.Infof("Starting server on %s", s.server.Addr)

		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			ch <- err
		}
	}()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		s.logger.Warn("Shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			_ = s.server.Close()

			return fmt.Errorf("error while gracefully shutting down server: %w", err)
		}

		s.logger.Info("Server was stopped")
	}

	return nil
}
