package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run() {
	port := getEnv("PORT", "8080")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealthz)

	addr := fmt.Sprintf(":%s", port)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "critical server error: %v\n", err)
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

func waitForShutdown(server *http.Server) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig
	fmt.Println("shutdown signal received, exiting...")
}

func getEnv(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if ok {
		return val
	}

	return fallback
}
