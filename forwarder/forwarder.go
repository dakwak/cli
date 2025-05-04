package forwarder

import (
	"log"
	"net"
	"io"

	"github.com/gorilla/websocket"
)


// HandleConnection bridges an existing WebSocket connection to a local TCP service.
func HandleConnection(wsConn *websocket.Conn, endpoint string) {
	localConn, err := net.Dial("tcp", endpoint)
	if err != nil {
	  log.Printf("Failed to connect to local service on %s: %v", endpoint, err)
	  return
	}
	defer localConn.Close()
	log.Printf("Connected to local service on %s", endpoint)

	// WebSocket to local TCP
	go func() {
	  for {
	    _, msg, err := wsConn.ReadMessage()
	    if err != nil {
	    	log.Printf("WebSocket read error: %v", err)
	    	localConn.Close()
	    	break
	    }
	    if _, err := localConn.Write(msg); err != nil {
	    	log.Printf("Local TCP write error: %v", err)
	    	break
	    }
	  }
	}()

	// Local TCP to WebSocket
	buf := make([]byte, 4096)
	for {
	  n, err := localConn.Read(buf)
	  if err != nil {
	    if err != io.EOF {
	      log.Printf("Local TCP read error: %v", err)
	    }
	    break
	  }
	  if err := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
	    log.Printf("WebSocket write error: %v", err)
	    break
	  }
	}
}

