package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alexey-y-a/bank-api/internal/config"
	"github.com/alexey-y-a/bank-api/internal/repository/postgres"
	"github.com/alexey-y-a/bank-api/pkg/logger"
)

func Run() {
	cfg := config.New()

	log, err := logger.New(cfg.GetLogLevel(), cfg.GetLogFormat())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "critical failed to init logger: %v\n", err)
	}

	logger.Info(log, "application starting", logger.Fields{
		"service":    "bank-api",
		"version":    "1.0.0",
		"host":       cfg.GetServerHost(),
		"port":       cfg.GetServerPort(),
		"log_level":  cfg.GetLogLevel(),
		"log_format": cfg.GetLogFormat(),
	})

	dbCfg := cfg.GetDatabaseConfig()

	db, err := postgres.NewDB(dbCfg)
	if err != nil {
		logger.Error(log, "failed to connect to database", err, nil)
		os.Exit(1)
	}

	defer db.Close()

	logger.Info(log, "connected to database", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = db.Ping(ctx)
	if err != nil {
		logger.Warn(log, "database ping failed on startup", err, nil)
	} else {
		logger.Debug(log, "database ping successful", nil)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealthz)

	addr := fmt.Sprintf("%s:%d", cfg.GetServerHost(), cfg.GetServerPort())

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	go func() {
		logger.Info(log, "server listening", logger.Fields{"address": addr})

		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Error(log, "critical server error", err, logger.Fields{"address": addr})
			os.Exit(1)
		}
	}()

	waitForShutdown(server)
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
