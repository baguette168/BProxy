package main

import (
	"flag"
	"log"

	"github.com/bproxy/bproxy/agent"
)

func main() {
	adminAddr := flag.String("admin", "127.0.0.1:8443", "Admin server address")
	flag.Parse()

	log.Printf("Starting BProxy Agent...")
	log.Printf("Connecting to admin at %s", *adminAddr)

	agentClient := agent.NewAgent(*adminAddr)
	if err := agentClient.Start(); err != nil {
		log.Fatalf("Agent error: %v", err)
	}
}