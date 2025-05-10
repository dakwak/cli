package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"cli/client"
	"cli/forwarder"
)

func main() {
	// Define CLI flags
  tunnelHost := flag.String("host", "tunnel.dakwak.com:443", "Tunnel server host (default: tunnel.dakwak.com:443)")
	token := flag.String("token", os.Getenv("DAKWAK_TOKEN"), "Auth token (or set DAKWAK_TOKEN env)")
	apikey := flag.String("apikey", "", "Optional: pass specific API key (client_id)")
	customHost := flag.String("local", "", "Optional: override local forwarding host (default: localhost)")

	flag.Parse()

	// Remaining args: mode and endpoint
	args := flag.Args()
	if len(args) < 2 {
		log.Fatalf("Usage: dakwak [--host tunnel.dakwak.com:443] [--token xxx] [--apikey client_id] [--local 127.0.0.1] http <host:port>")
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

	// Determine local target for forwarding
	localTarget := endpoint
	if *customHost != "" {
		localTarget = fmt.Sprintf("%s:%s", *customHost, strings.Split(endpoint, ":")[1])
	}

	log.Printf("Connecting to tunnel (%s) for local service %s", *tunnelHost, localTarget)

	// Establish tunnel
	wsConn, clientID, err := client.ConnectTunnel(*tunnelHost, *token, *apikey, localTarget)
        if err != nil {
        	log.Fatalf("Failed to connect to tunnel: %v", err)
        }
        fmt.Printf("Public URL: https://%s.tunnel.dakwak.com\n", clientID)
        wsConn.SetReadLimit(5 * 1024 * 1024)
	if err != nil {
		log.Fatalf("Failed to connect to tunnel: %v", err)
	}

	forwarder.HandleConnection(wsConn, localTarget)
}

