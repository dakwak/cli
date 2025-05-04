package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"cli/client"
	"cli/forwarder"
)

func main() {
	// Define flags
	tunnelHost := flag.String("host", "tunnel.dakwak.com:443", "Tunnel server host (default: tunnel.dakwak.com:443)")
	token := flag.String("token", os.Getenv("DAKWAK_TOKEN"), "Auth token (default: from DAKWAK_TOKEN env)")
	flag.Parse()

	// Remaining args: mode and endpoint
	args := flag.Args()
	if len(args) < 2 {
		log.Fatalf("Usage: dakwak [--host tunnel.dakwak.com:443] [--token xxx] http <host:port>")
	}

	mode := args[0]
	endpoint := args[1]

	if mode != "http" {
		log.Fatalf("Unsupported mode: %s", mode)
	}

	if !strings.Contains(endpoint, ":") {
		log.Fatalf("Invalid endpoint format. Must be host:port")
	}

	if *token == "" {
		log.Fatal("Missing token. Provide --token or set DAKWAK_TOKEN environment variable")
	}

	log.Printf("Connecting to tunnel (%s) for %s", *tunnelHost, endpoint)

	wsConn, err := client.ConnectTunnel(*token)
	if err != nil {
		log.Fatalf("Failed to connect to tunnel: %v", err)
	}

	forwarder.HandleConnection(wsConn, endpoint)
}

