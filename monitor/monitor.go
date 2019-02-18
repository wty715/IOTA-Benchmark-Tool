package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"
	"fmt"
	"auto-wallet/monitor/transactions"
)

func Shutdown(timeout time.Duration) {
	select {
	case <-time.After(timeout):
	}
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// print out current zmq version
	major, minor, patch := zmq4.Version()
	fmt.Printf("running ZMQ %d.%d.%d\n", major, minor, patch)

	// start feeds
	go startTxFeed()
	go startMilestoneFeed()
	go startConfirmationFeed()

	select {
	case <-sigs:
		Shutdown(time.Duration(1500) * time.Millisecond)
	}
}