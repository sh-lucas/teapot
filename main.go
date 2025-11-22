package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sh-lucas/teapot/cup"
	"github.com/sh-lucas/teapot/cup/router"
	"github.com/sh-lucas/teapot/handlers/logs"
)

func main() {
	// Run the server in a goroutine so we can handle signals
	go router.Route(cup.PORT)

	// Wait for interrupt signal to gracefully shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	
	fmt.Println("Server started. Press Ctrl+C to stop.")
	<-stop

	fmt.Println("\nShutting down...")
	logs.Shutdown()
	fmt.Println("Goodbye!")
}