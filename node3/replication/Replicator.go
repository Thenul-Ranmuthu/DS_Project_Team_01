package replication

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/DS_node/config"
	"github.com/DS_node/models"
)

// ReplicationResult stores the outcome of a replication attempt to a specific peer
type ReplicationResult struct {
	PeerURL string
	Success bool
	Error   error
}

// ReplicateToPeers sends the uploaded file and metadata to all other nodes in the cluster
// It returns a slice of results for each peer.
func ReplicateToPeers(filePath string, fileName string, userID uint, originalName string, mimeType string, fileSize int64) []ReplicationResult {
	cfg := config.Load()
	if len(cfg.Peers) == 0 {
		fmt.Println("[Replicator] No peers configured. Skipping upload replication.")
		return nil
	}

	results := make([]ReplicationResult, len(cfg.Peers))
	var wg sync.WaitGroup

	for i, peer := range cfg.Peers {
		wg.Add(1)
		go func(idx int, peerURL string) {
			defer wg.Done()
			
			res := ReplicationResult{PeerURL: peerURL, Success: false}
			
			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("[Replicator] Error opening local file: %v\n", err)
				res.Error = err
				results[idx] = res
				return
			}
			defer file.Close()

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			part, err := writer.CreateFormFile("file", fileName)
			if err != nil {
				fmt.Printf("[Replicator] Error creating form file: %v\n", err)
				res.Error = err
				results[idx] = res
				return
			}
			io.Copy(part, file)

			// Add metadata as form fields
			writer.WriteField("user_id", fmt.Sprintf("%d", userID))
			writer.WriteField("original_name", originalName)
			writer.WriteField("mime_type", mimeType)
			writer.WriteField("file_size", fmt.Sprintf("%d", fileSize))

			writer.Close()

			targetURL := fmt.Sprintf("%s/internal/replicate", peerURL)

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Post(targetURL, writer.FormDataContentType(), body)
			if err != nil {
				fmt.Printf("[Replicator] Failed to reach peer %s: %v\n", peerURL, err)
				res.Error = err
				results[idx] = res
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("[Replicator] Successfully replicated %s to %s\n", fileName, peerURL)
				res.Success = true
			} else {
				fmt.Printf("[Replicator] Peer %s refused replication: status %d\n", peerURL, resp.StatusCode)
				res.Error = fmt.Errorf("peer returned status %d", resp.StatusCode)
				
				// Queue for retry
				meta := map[string]interface{}{
					"file_name":     fileName,
					"user_id":       userID,
					"original_name": originalName,
					"mime_type":     mimeType,
					"file_size":     fileSize,
				}
				metaJSON, _ := json.Marshal(meta)
				AddToQueue(models.ReplicateFileUpload, peerURL, string(metaJSON), filePath)
			}
			results[idx] = res
		}(i, peer)
	}

	wg.Wait()
	return results
}

// ReplicateDeleteToPeers sends a DELETE request to all configured backup nodes
func ReplicateDeleteToPeers(fileName string) []ReplicationResult {
	cfg := config.Load()

	if len(cfg.Peers) == 0 {
		fmt.Println("[Replicator] No peers configured for deletion broadcast.")
		return nil
	}

	results := make([]ReplicationResult, len(cfg.Peers))
	var wg sync.WaitGroup

	for i, peer := range cfg.Peers {
		wg.Add(1)
		go func(idx int, peerURL string) {
			defer wg.Done()
			res := ReplicationResult{PeerURL: peerURL, Success: false}

			// Construct the internal delete URL
			deleteURL := fmt.Sprintf("%s/internal/delete/%s", peerURL, fileName)

			req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
			if err != nil {
				fmt.Printf("[Replicator] Failed to create delete request for %s: %v\n", peerURL, err)
				res.Error = err
				results[idx] = res
				return
			}

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("[Replicator] Peer %s unreachable for deletion: %v\n", peerURL, err)
				res.Error = err
				results[idx] = res
				
				// Queue for retry
				AddToQueue(models.ReplicateFileDelete, peerURL, fileName, "")
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("[Replicator] Successfully deleted replica %s from %s\n", fileName, peerURL)
				res.Success = true
			} else {
				fmt.Printf("[Replicator] Peer %s failed to delete: status %d\n", peerURL, resp.StatusCode)
				res.Error = fmt.Errorf("peer returned status %d", resp.StatusCode)
				
				// Queue for retry
				AddToQueue(models.ReplicateFileDelete, peerURL, fileName, "")
			}
			results[idx] = res
		}(i, peer)
	}

	wg.Wait()
	return results
}
