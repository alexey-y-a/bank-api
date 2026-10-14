package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alexey-y-a/bank-api/internal/config"
	cardcrypto "github.com/alexey-y-a/bank-api/internal/crypto"
	accounthandler "github.com/alexey-y-a/bank-api/internal/handler/account"
	cardhandler "github.com/alexey-y-a/bank-api/internal/handler/card"
	credithandler "github.com/alexey-y-a/bank-api/internal/handler/credit"
	transferhandler "github.com/alexey-y-a/bank-api/internal/handler/transfer"
	userhandler "github.com/alexey-y-a/bank-api/internal/handler/user"
	"github.com/alexey-y-a/bank-api/internal/kafka"
	"github.com/alexey-y-a/bank-api/internal/middleware"
	"github.com/alexey-y-a/bank-api/internal/probe"
	"github.com/alexey-y-a/bank-api/internal/repository/postgres"
	goredis "github.com/alexey-y-a/bank-api/internal/repository/redis"
	accountservice "github.com/alexey-y-a/bank-api/internal/service/account"
	cardservice "github.com/alexey-y-a/bank-api/internal/service/card"
	creditservice "github.com/alexey-y-a/bank-api/internal/service/credit"
	"github.com/alexey-y-a/bank-api/internal/service/scheduler"
	transferservice "github.com/alexey-y-a/bank-api/internal/service/transfer"
	userservice "github.com/alexey-y-a/bank-api/internal/service/user"
	"github.com/alexey-y-a/bank-api/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context) {
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

	pingCtx, cancelPing := context.WithTimeout(ctx, 3*time.Second)
	err = db.Ping(pingCtx)
	cancelPing()

	if err != nil {
		logger.Warn(log, "database ping failed on startup", err, nil)
	} else {
		logger.Debug(log, "database ping succeeded", nil)
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

	kafkaProducer, err := kafka.NewProducer(cfg.GetKafkaBrokers())
	if err != nil {
		logger.Error(log, "failed to create kafka producer", err, nil)
		os.Exit(1)
	}

	defer kafkaProducer.Close()

	kafkaCtx, kafkaCancel := context.WithTimeout(ctx, 3*time.Second)
	err = kafkaProducer.Ping(kafkaCtx)
	kafkaCancel()

	if err != nil {
		logger.Warn(log, "kafka ping failed on startup", err, nil)
	} else {
		logger.Info(log, "kafka producer initialized", logger.Fields{"brokers": cfg.GetKafkaBrokers()})
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

	transferRepo := postgres.NewTransferRepo(db.Pool())
	transferSvc := transferservice.NewService(accountRepo, transferRepo, db.Pool(), kafkaProducer, log)
	transferHdl := transferhandler.NewHandler(transferSvc)

	creditRepo := postgres.NewCreditRepo(db.Pool())
	creditSvc := creditservice.NewService(creditRepo, accountRepo)
	creditHdl := credithandler.NewHandler(creditSvc)

	creditScheduler := scheduler.NewScheduler(creditRepo, log)

	authMW := middleware.Auth([]byte(cfg.GetJWTSecret()))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /readyz", readyProbe.Handler())
	mux.HandleFunc("GET /metrics", promhttp.Handler().ServeHTTP)
	mux.HandleFunc("POST /register", userHdl.Register)
	mux.HandleFunc("POST /login", userHdl.Login)

	mux.Handle("GET /accounts", authMW(http.HandlerFunc(accountHdl.GetUserAccounts)))
	mux.Handle("POST /accounts", authMW(http.HandlerFunc(accountHdl.CreateAccount)))
	mux.Handle("GET /accounts/{id}", authMW(http.HandlerFunc(accountHdl.GetAccount)))
	mux.Handle("POST /accounts/{id}/deposit", authMW(http.HandlerFunc(accountHdl.Deposit)))
	mux.Handle("POST /accounts/{id}/withdraw", authMW(http.HandlerFunc(accountHdl.Withdraw)))

	mux.Handle("POST /cards", authMW(http.HandlerFunc(cardHdl.CreateCard)))
	mux.Handle("GET /cards", authMW(http.HandlerFunc(cardHdl.GetUserCards)))
	mux.Handle("POST /cards/{id}/block", authMW(http.HandlerFunc(cardHdl.BlockCard)))
	mux.Handle("POST /cards/{id}/pay", authMW(http.HandlerFunc(cardHdl.PayWithCard)))

	mux.Handle("POST /transfer", authMW(http.HandlerFunc(transferHdl.MakeTransfer)))
	mux.Handle("GET /history", authMW(http.HandlerFunc(transferHdl.GetHistory)))

	mux.Handle("POST /credits", authMW(http.HandlerFunc(creditHdl.CreateCredit)))
	mux.Handle("GET /credits/{id}/schedule", authMW(http.HandlerFunc(creditHdl.GetSchedule)))
	mux.Handle("POST /credits/{id}/pay", authMW(http.HandlerFunc(creditHdl.MakePayment)))

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

	errGr, egCtx := errgroup.WithContext(ctx)

	errGr.Go(func() error {
		logger.Info(log, "server listening", logger.Fields{"address": addr})

		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("http server listen: %w", err)
	})

	errGr.Go(func() error {
		creditScheduler.Run(egCtx)
		creditScheduler.Stop()

		return nil
	})

	errGr.Go(func() error {
		return shutdownServer(egCtx, log, server)
	})

	err = errGr.Wait()
	if err != nil {
		logger.Error(log, "critical application error", err, nil)
		os.Exit(1)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
