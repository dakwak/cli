package client

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

var writeMu sync.Mutex

func ConnectTunnel(tunnelHost, token, apikey, host string) (*websocket.Conn, error) {
	if tunnelHost == "" {
		tunnelHost = os.Getenv("DAKWAK_TUNNEL_HOST")
		if tunnelHost == "" {
			tunnelHost = "tunnel.dakwak.com:443"
		}
	}

	// Build query parameters
	query := url.Values{}
	query.Set("token", token)
	if apikey != "" {
		query.Set("apikey", apikey)
	}
	if host != "" {
		query.Set("host", host)
	}

	u := url.URL{
		Scheme:   "wss",
		Host:     tunnelHost,
		Path:     "/connect",
		RawQuery: query.Encode(),
	}

	log.Printf("Dialing tunnel: %s", u.String())

	dialer := websocket.Dialer{
		EnableCompression: false,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tunnel: %w", err)
	}
	conn.SetReadLimit(5 * 1024 * 1024)

	log.Println("Tunnel connection established")
	return conn, nil
}

// SafeWrite writes data to WebSocket safely (mutex locked)
func SafeWrite(conn *websocket.Conn, data []byte) error {
	writeMu.Lock()
	defer writeMu.Unlock()

	writer, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	return writer.Close()
}

