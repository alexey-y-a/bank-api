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

func waitForShutdown(server *http.Server) {
	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("received shutdown signal, starting graceful shutdown...")

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "graceful shutdown error: %v\n", err)
	}

	fmt.Println("server stopped successfully")
}
