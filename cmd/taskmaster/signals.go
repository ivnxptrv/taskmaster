package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"taskmaster/internal/gateway"
)

type SignalWatcher struct {
	signal chan os.Signal
	g      gateway.Gateway
}

func (w *SignalWatcher) listen() {
	sig := <-w.signal
	switch sig {
	case os.Interrupt, syscall.SIGTERM:
		fmt.Println("Shutdown request received.")
		w.g.Shutdown()
		os.Exit(0)
	case syscall.SIGHUP:
		fmt.Println("Reloading configuration...")
		// w.g.Reload()
	default:
		fmt.Printf("Received signal: %v. No specific handler.\n", sig)
	}
}

func NewSignalWatcher(g gateway.Gateway) *SignalWatcher {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	return &SignalWatcher{
		signal: sigChan,
		g:      g,
	}
}
