package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"pushly/internal/relay"
)

func main() {
	port := flag.Int("port", 8080, "HTTP port to listen on")
	flag.Parse()

	server := relay.NewRelayServer()
	addr := fmt.Sprintf(":%d", *port)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	log.Printf("Pushly Relay Server listening on http://localhost:%d\n", *port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
