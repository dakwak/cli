package forwarder

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
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

const MaxChunkSize = 25 * 1024 * 1024 // 5MB

func HandleConnection(wsConn *websocket.Conn, endpoint string) {
	log.Printf("Listening for HTTP relay requests to %s", endpoint)

	var mu sync.Mutex
	var closed bool

	wsConn.SetCloseHandler(func(code int, text string) error {
		log.Printf("WebSocket closed by server: %d - %s", code, text)
		mu.Lock()
		closed = true
		mu.Unlock()
		return nil
	})

	safeWrite := func(data []byte) {
		mu.Lock()
		defer mu.Unlock()

		if closed {
			log.Println("Skipping write, connection already closed")
			return
		}

		if !json.Valid(data) {
			log.Println("Skipping invalid JSON write")
			closed = true
			return
		}

		if len(data) > MaxChunkSize {
			chunks := splitIntoChunks(data, MaxChunkSize)
			for i, chunk := range chunks {
				frame := map[string]interface{}{
					"chunk": i,
					"total": len(chunks),
					"data":  chunk,
				}
				chunkJSON, _ := json.Marshal(frame)
				if err := wsConn.WriteMessage(websocket.TextMessage, chunkJSON); err != nil {
					log.Printf("safe write chunk error: %v", err)
					closed = true
					return
				}
			}
			return
		}

		if err := wsConn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("safe write error: %v", err)
			closed = true
			return
		}
	}

	for {
		mu.Lock()
		if closed {
			log.Println("Connection previously closed, exiting handler loop")
			mu.Unlock()
			break
		}
		mu.Unlock()

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
			safeWrite([]byte(`{"status":500,"body":"Failed to create request"}`))
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

		jsonResp, err := json.Marshal(response)
		if err != nil {
			log.Printf("JSON marshal error: %v", err)
			safeWrite([]byte(`{"status":500,"body":"Tunnel marshal error"}`))
			continue
		}

		log.Printf("Response for [%s %s] → %d, size: %d bytes", req.Method, req.Path, response.Status, len(jsonResp))
		log.Printf("Client sending response JSON: SHA256=%x", sha256.Sum256(jsonResp))
		safeWrite(jsonResp)
	}
}

func splitIntoChunks(data []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

