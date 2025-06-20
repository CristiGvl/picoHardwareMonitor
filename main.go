package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/CristiGvl/picoHWMon/api"
	"github.com/CristiGvl/picoHWMon/internal/platform"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8080", "Port to run the server on")
	bind := flag.String("bind", "0.0.0.0", "IP address to bind the server to")
	flag.Parse()

	// Check platform support
	// Validate platform support
	if err := platform.ValidateSupport(); err != nil {
		log.Fatalf("Platform validation failed: %v", err)
	}

	// Create and start the API server
	server, err := api.NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		if err := server.Shutdown(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		os.Exit(0)
	}()

	// Start the server
	log.Printf("Starting picoHWMon server on %s:%s", *bind, *port)
	log.Fatal(server.Start(*bind + ":" + *port))
}
