package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func waitForShutdown(server *http.Server, cancelScheduler context.CancelFunc, stopScheduler func()) {
	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	fmt.Println("received shutdown signal, starting gracefull shutdown...")

	cancelScheduler()
	stopScheduler()
	fmt.Println("credit scheduler stopped")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "graceful shutdown error: %v\n", err)
	}

	fmt.Println("server stopped successfully")
}
