package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alexey-y-a/bank-api/pkg/logger"
	"github.com/sirupsen/logrus"
)

func shutdownServer(ctx context.Context, log *logrus.Logger, server *http.Server) error {
	<-ctx.Done()

	logger.Info(log, "shutdown signal received, stopping server", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		return fmt.Errorf("graceful shutdown server: %w", err)
	}

	logger.Info(log, "server stopped successfully", nil)

	return nil
}
