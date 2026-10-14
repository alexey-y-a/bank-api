package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/alexey-y-a/bank-api/internal/app"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	app.Run(ctx)
}
