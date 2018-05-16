package main

import (
	"os"
	"os/signal"
	"syscall"
)

func onShutdown(f func()) {
	sigc := make(chan os.Signal, 3)
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		<-sigc
		f()
		os.Exit(1)
	}()
}
