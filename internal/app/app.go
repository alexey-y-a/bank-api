package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alexey-y-a/bank-api/internal/config"
	cardcrypto "github.com/alexey-y-a/bank-api/internal/crypto"
	accounthandler "github.com/alexey-y-a/bank-api/internal/handler/account"
	cardhandler "github.com/alexey-y-a/bank-api/internal/handler/card"
	userhandler "github.com/alexey-y-a/bank-api/internal/handler/user"
	"github.com/alexey-y-a/bank-api/internal/middleware"
	"github.com/alexey-y-a/bank-api/internal/probe"
	"github.com/alexey-y-a/bank-api/internal/repository/postgres"
	goredis "github.com/alexey-y-a/bank-api/internal/repository/redis"
	accountservice "github.com/alexey-y-a/bank-api/internal/service/account"
	cardservice "github.com/alexey-y-a/bank-api/internal/service/card"
	userservice "github.com/alexey-y-a/bank-api/internal/service/user"
	"github.com/alexey-y-a/bank-api/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	readyProbe := probe.NewReadyProbe()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = db.Ping(ctx)
	if err != nil {
		logger.Warn(log, "database ping failed on startup", err, nil)
	} else {
		logger.Debug(log, "database ping successful", nil)
		readyProbe.MarkReady()
	}

	var userCache *goredis.UserCache

	redisAddr := cfg.GetRedisAddr()
	redisPass := cfg.GetRedisPass()

	if redisAddr != "" {
		userCache = goredis.NewUserCache(redisAddr, redisPass)
		logger.Info(log, "redis cache initialized", logger.Fields{"addr": redisAddr})
	} else {
		logger.Warn(log, "redis not configured - cache disabled", nil, nil)
	}

	userRepo := postgres.NewUserRepository(db.Pool())
	userSvc := userservice.NewService(userRepo, userCache, cfg.GetJWTSecret(), cfg.GetJWTTTLHours())
	userHdl := userhandler.NewHandler(userSvc)

	accountRepo := postgres.NewAccountRepository(db.Pool())
	accountSvc := accountservice.NewService(accountRepo)
	accountHdl := accounthandler.NewHandler(accountSvc)

	cardEncryptor := cardcrypto.NewCardEncryptor(cfg.GetCardAESSecret(), cfg.GetHMACSecret())
	cardRepo := postgres.NewCardRepository(db.Pool())
	cardSvc := cardservice.NewService(cardEncryptor, cardRepo, accountRepo)
	cardHdl := cardhandler.NewHandler(cardSvc)

	authMW := middleware.Auth([]byte(cfg.GetJWTSecret()))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /readyz", readyProbe.Handler())
	mux.HandleFunc("GET /metrics", promhttp.Handler().ServeHTTP)
	mux.HandleFunc("POST/register", userHdl.Register)
	mux.HandleFunc("POST/login", userHdl.Login)

	mux.Handle("GET /accounts", authMW(http.HandlerFunc(accountHdl.GetUserAccounts)))
	mux.Handle("POST /accounts", authMW(http.HandlerFunc(accountHdl.CreateAccount)))
	mux.Handle("GET /accounts/{id}", authMW(http.HandlerFunc(accountHdl.GetAccount)))
	mux.Handle("POST /accounts/{id}/deposit", authMW(http.HandlerFunc(accountHdl.Deposit)))
	mux.Handle("POST /accounts/{id}/withdraw", authMW(http.HandlerFunc(accountHdl.Withdraw)))

	mux.Handle("POST /cards", authMW(http.HandlerFunc(cardHdl.CreateCard)))
	mux.Handle("GET /cards", authMW(http.HandlerFunc(cardHdl.GetUserCards)))
	mux.Handle("POST /cards/{id}/block", authMW(http.HandlerFunc(cardHdl.BlockCard)))
	mux.Handle("POST /cards/{id}/pay", authMW(http.HandlerFunc(cardHdl.PayWithCard)))

	var handler http.Handler = mux
	handler = middleware.Recover(log)(handler)
	handler = middleware.Logging(log)(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.Metrics(handler)

	addr := fmt.Sprintf("%s:%d", cfg.GetServerHost(), cfg.GetServerPort())

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
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
