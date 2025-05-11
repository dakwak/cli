package forwarder

import (
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

const StreamChunkSize = 128 * 1024 // 128 KB

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

	writeChunk := func(data []byte) error {
		writer, err := wsConn.NextWriter(websocket.BinaryMessage)
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		if err != nil {
			writer.Close()
			return err
		}
		return writer.Close()
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

		var req TunnelRequest
		if err := json.NewDecoder(r).Decode(&req); err != nil {
			log.Printf("Forwarder Invalid tunnel JSON message: %v", err)
			continue
		}

		fullURL := "http://" + endpoint + "/" + strings.TrimPrefix(req.Path, "/")
		httpReq, err := http.NewRequest(req.Method, fullURL, strings.NewReader(req.Body))
		if err != nil {
			log.Printf("Failed to create HTTP request: %v", err)
			writeChunk([]byte(`{"status":500,"body":"Failed to create request"}`))
			continue
		}
		for k, v := range req.Headers {
			httpReq.Header.Set(k, v)
		}

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			log.Printf("Local HTTP error: %v", err)
			writeChunk([]byte(`{"status":502,"body":"Bad Gateway"}`))
			continue
		}
		defer resp.Body.Close()

		headers := make(map[string]string)
		for k, v := range resp.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		// Send metadata with Stream flag
		respMeta := struct {
			Status  int               `json:"status"`
			Headers map[string]string `json:"headers"`
			Stream  bool              `json:"stream"`
		}{
			Status:  resp.StatusCode,
			Headers: headers,
			Stream:  true,
		}

		metaBytes, err := json.Marshal(respMeta)
		if err != nil {
			log.Printf("JSON marshal error: %v", err)
			writeChunk([]byte(`{"status":500,"body":"Tunnel marshal error"}`))
			continue
		}

		if err := writeChunk(metaBytes); err != nil {
			log.Printf("WebSocket write error (meta): %v", err)
			continue
		}

		// Stream body in chunks
		buf := make([]byte, StreamChunkSize)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if err := writeChunk(buf[:n]); err != nil {
					log.Printf("WebSocket write error (body): %v", err)
					break
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("HTTP body read error: %v", err)
				break
			}
		}

		// Signal end of stream
		_ = writeChunk([]byte("_#_END_#_"))

		log.Printf("Response for [%s %s] → %d", req.Method, req.Path, resp.StatusCode)
	}
}

