package client

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
)

func ConnectTunnel(token string) (*websocket.Conn, error) {
  host := os.Getenv("DAKWAK_TUNNEL_HOST")
  if host == "" {
    host = "tunnel.dakwak.com:443"
  }
  
  u := url.URL{
    Scheme:   "wss",
    Host:     host,
    Path:     "/connect",
    RawQuery: "token=" + token,
  }
  
  log.Printf("Dialing tunnel: %s", u.String())
  
  conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
  if err != nil {
    return nil, fmt.Errorf("failed to connect to tunnel: %w", err)
  }
  
  log.Println("Tunnel connection established")
  return conn, nil
}

