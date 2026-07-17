// Command horus manages PK-range partitions: analyzes layout, plans
// time-aligned boundary changes, and applies them.
//
// main stays minimal: it only calls internal/cli and exits with its code.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/RTolkachev/horus/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(cli.Run(ctx, os.Args[1:]))
}
