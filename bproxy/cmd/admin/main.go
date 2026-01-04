package main

import (
	"flag"
	"log"

	"github.com/bproxy/bproxy/admin"
)

func main() {
	addr := flag.String("addr", "0.0.0.0:8443", "Admin server listen address")
	certFile := flag.String("cert", "", "TLS certificate file (optional)")
	keyFile := flag.String("key", "", "TLS key file (optional)")
	flag.Parse()

	log.Printf("Starting BProxy Admin Server...")

	adminServer, err := admin.NewAdmin(*addr, *certFile, *keyFile)
	if err != nil {
		log.Fatalf("Failed to create admin server: %v", err)
	}

	if err := adminServer.Start(); err != nil {
		log.Fatalf("Admin server error: %v", err)
	}
}