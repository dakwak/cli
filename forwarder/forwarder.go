package forwarder

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"log"
	"net/http"
        "sync"
	"github.com/gorilla/websocket"
)

type TunnelRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

type TunnelResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body"`
}
func HandleConnection(wsConn *websocket.Conn, endpoint string) {
	log.Printf("Listening for HTTP relay requests to %s", endpoint)

	var mu sync.Mutex
	var closed bool

  safeWrite := func(data []byte) {
  	mu.Lock()
  	defer mu.Unlock()
  
  	if closed {
  		log.Println("Skipping write, connection already closed")
  		return
  	}
  
  	writer, err := wsConn.NextWriter(websocket.TextMessage)
  	if err != nil {
  		log.Printf("safe write error (NextWriter): %v", err)
  		closed = true
  		return
  	}
  
  	_, copyErr := io.Copy(writer, bytes.NewReader(data))
  	closeErr := writer.Close()
  
  	if copyErr != nil {
  		log.Printf("safe write error (copy): %v", copyErr)
  		closed = true
  	}
  	if closeErr != nil {
  		log.Printf("safe write error (close): %v", closeErr)
  		closed = true
  	}
  }


	for {
		mt, r, err := wsConn.NextReader()
		if err != nil {
			log.Printf("Forwarder WebSocket read error: %v", err)
			break
		}
		if mt != websocket.TextMessage {
			log.Printf("Forwarder Unexpected WebSocket message type: %d", mt)
			continue
		}

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			log.Printf("Forwarder Error reading WebSocket message: %v", err)
			break
		}

		var req TunnelRequest
		if err := json.Unmarshal(buf.Bytes(), &req); err != nil {
			log.Printf("Forwarder Invalid tunnel JSON message: %v", err)
			continue
		}

		fullURL := "http://" + endpoint + "/" + strings.TrimPrefix(req.Path, "/")
		httpReq, err := http.NewRequest(req.Method, fullURL, bytes.NewBufferString(req.Body))
		if err != nil {
			log.Printf("Failed to create HTTP request: %v", err)
			continue
		}
		for k, v := range req.Headers {
			httpReq.Header.Set(k, v)
		}

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			log.Printf("Local HTTP error: %v", err)
			safeWrite([]byte(`{"status":502,"body":"Bad Gateway"}`))
			continue
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		headers := make(map[string]string)
		for k, v := range resp.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		response := TunnelResponse{
			Status:  resp.StatusCode,
			Headers: headers,
			Body:    string(bodyBytes),
		}

		jsonResp, _ := json.Marshal(response)
		log.Printf("Client sending response JSON (len=%d): SHA256=%x", len(jsonResp), sha256.Sum256(jsonResp))
		safeWrite(jsonResp)
	}
}

