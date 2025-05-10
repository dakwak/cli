package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

var writeMu sync.Mutex

func ConnectTunnel(tunnelHost, token, apikey, host string) (*websocket.Conn, string, error) {
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

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to connect to tunnel: %w", err)
	}
	conn.SetReadLimit(5 * 1024 * 1024)

	// Read the initial client_id JSON
	_, msg, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, "", fmt.Errorf("failed to read client_id from server: %w", err)
	}

	var init struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(msg, &init); err != nil {
		conn.Close()
		return nil, "", fmt.Errorf("invalid client_id message: %w", err)
	}

	if init.ClientID == "" {
		conn.Close()
		return nil, "", fmt.Errorf("client_id missing in tunnel response")
	}

	log.Println("Tunnel connection established")
	return conn, init.ClientID, nil
}

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

