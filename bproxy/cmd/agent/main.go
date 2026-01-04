package main

import (
        "flag"
        "log"

        "github.com/bproxy/bproxy/agent"
)

func main() {
        adminAddr := flag.String("admin", "127.0.0.1:8443", "Admin server address")
        cascadePort := flag.Int("cascade", 0, "Port for cascade connections (0 = disabled)")
        flag.Parse()

        log.Printf("Starting BProxy Agent...")
        log.Printf("Connecting to admin at %s", *adminAddr)
        if *cascadePort > 0 {
                log.Printf("Cascade mode enabled on port %d", *cascadePort)
        }

        agentClient := agent.NewAgent(*adminAddr, *cascadePort)
        if err := agentClient.Start(); err != nil {
                log.Fatalf("Agent error: %v", err)
        }
}