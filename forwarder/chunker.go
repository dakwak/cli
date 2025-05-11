package forwarder
import (
	"encoding/json"
	"log"
	"math"
	"sync"
)

type ChunkedMessage struct {
	ClientID    string `json:"client_id"`
	ChunkIndex  int    `json:"chunk_index"`
	TotalChunks int    `json:"total_chunks"`
	IsLast      bool   `json:"is_last"`
	Payload     string `json:"payload"`
}

const MaxChunkSize = 5 * 1024 * 1024 // 5MB

// SplitLargeResponse breaks down a JSON blob into multiple WebSocket-safe chunks.
func SplitLargeResponse(clientID string, jsonData []byte) [][]byte {
	var chunks [][]byte
	totalChunks := int(math.Ceil(float64(len(jsonData)) / float64(MaxChunkSize)))

	for i := 0; i < totalChunks; i++ {
		start := i * MaxChunkSize
		end := start + MaxChunkSize
		if end > len(jsonData) {
			end = len(jsonData)
		}

		part := jsonData[start:end]
		chunk := ChunkedMessage{
			ClientID:    clientID,
			ChunkIndex:  i,
			TotalChunks: totalChunks,
			IsLast:      i == totalChunks-1,
			Payload:     string(part),
		}

		chunkBytes, err := json.Marshal(chunk)
		if err != nil {
			log.Printf("Failed to marshal chunk %d: %v", i, err)
			continue
		}
		chunks = append(chunks, chunkBytes)
	}

	return chunks
}

// Reassembler for incoming chunks from client
var (
	chunkBuffers = make(map[string][]string)
	chunkLocks   = make(map[string]*sync.Mutex)
)

func AddChunk(clientID string, chunk ChunkedMessage) (string, bool) {
	if chunkLocks[clientID] == nil {
		chunkLocks[clientID] = &sync.Mutex{}
	}
	chunkLocks[clientID].Lock()
	defer chunkLocks[clientID].Unlock()

	chunkBuffers[clientID] = append(chunkBuffers[clientID], chunk.Payload)

	if chunk.IsLast {
		full := ""
		for _, part := range chunkBuffers[clientID] {
			full += part
		}
		delete(chunkBuffers, clientID)
		return full, true
	}
	return "", false
}
