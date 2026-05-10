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
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "critical failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "critical failed to init logger: %v\n", err)
	}

	logger.Info(log, "application starting", logger.Fields{
		"service":    "bank-api",
		"version":    "1.0.0",
		"host":       cfg.Server.Host,
		"port":       cfg.Server.Port,
		"log_level":  cfg.Log.Level,
		"log_format": cfg.Log.Format,
	})

	db, err := postgres.NewDB(cfg.Database)
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

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

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
