package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/xgerman/mongofuse/internal/mongofuse/cmd"
)

func main() {
	// Handle signals for clean shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nShutting down...")
		os.Exit(0)
	}()

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
