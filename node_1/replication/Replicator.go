package replication

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/DS_node/config"
)

// ReplicateToPeers sends the uploaded file to all other nodes in the cluster
func ReplicateToPeers(filePath string, fileName string) {
	cfg := config.Load()

	if len(cfg.Peers) == 0 {
		fmt.Println("[Replicator] No peers found in configuration. Skipping replication.")
		return
	}

	for _, peer := range cfg.Peers {
		go func(peerURL string) {
			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("[Replicator] Error opening local file %s: %v\n", filePath, err)
				return
			}
			defer file.Close()

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			part, err := writer.CreateFormFile("file", fileName)
			if err != nil {
				fmt.Printf("[Replicator] Failed to create form file: %v\n", err)
				return
			}

			_, err = io.Copy(part, file)
			if err != nil {
				fmt.Printf("[Replicator] Failed to copy file content: %v\n", err)
				return
			}
			writer.Close()

			targetURL := fmt.Sprintf("%s/internal/replicate", peerURL)

			client := &http.Client{
				Timeout: 10 * time.Second,
			}

			resp, err := client.Post(targetURL, writer.FormDataContentType(), body)
			if err != nil {
				fmt.Printf("[Replicator] Failed to reach peer %s: %v\n", peerURL, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("[Replicator] Successfully replicated %s to %s\n", fileName, peerURL)
			} else {
				fmt.Printf("[Replicator] Peer %s returned status: %d\n", peerURL, resp.StatusCode)
			}
		}(peer)
	}
}

// ReplicateDeleteToPeers informs all peers to delete a specific file replica
func ReplicateDeleteToPeers(fileName string) {
	cfg := config.Load()

	if len(cfg.Peers) == 0 {
		fmt.Println("[Replicator] No peers found for deletion broadcast.")
		return
	}

	for _, peer := range cfg.Peers {
		go func(peerURL string) {
			// Construct the DELETE URL, e.g., http://localhost:5051/internal/delete/filename.png
			targetURL := fmt.Sprintf("%s/internal/delete/%s", peerURL, fileName)

			req, err := http.NewRequest(http.MethodDelete, targetURL, nil)
			if err != nil {
				fmt.Printf("[Replicator] Failed to create delete request for %s: %v\n", peerURL, err)
				return
			}

			// Using a slightly shorter timeout for deletions
			client := &http.Client{
				Timeout: 5 * time.Second,
			}

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("[Replicator] Failed to reach peer %s for deletion: %v\n", peerURL, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("[Replicator] Successfully deleted replica %s from %s\n", fileName, peerURL)
			} else {
				fmt.Printf("[Replicator] Peer %s failed to delete: status %d\n", peerURL, resp.StatusCode)
			}
		}(peer)
	}
}
