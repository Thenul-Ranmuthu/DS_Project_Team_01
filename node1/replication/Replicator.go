package replication

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/DS_node/config"
)

// ReplicateToPeers sends the uploaded file and metadata to all other nodes in the cluster
func ReplicateToPeers(filePath string, fileName string, userID uint, originalName string, mimeType string, fileSize int64, lamportClock uint64) {
	cfg := config.Load()
	if len(cfg.Peers) == 0 {
		fmt.Println("[Replicator] No peers configured. Skipping upload replication.")
		return
	}

	for _, peer := range cfg.Peers {
		go func(peerURL string) {
			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("[Replicator] Error opening local file: %v\n", err)
				return
			}
			defer file.Close()

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			part, err := writer.CreateFormFile("file", fileName)
			if err != nil {
				fmt.Printf("[Replicator] Error creating form file: %v\n", err)
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

			req, err := http.NewRequest(http.MethodPost, targetURL, body)
			if err != nil {
				fmt.Printf("[Replicator] Error creating request: %v\n", err)
				return
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("X-Lamport-Clock", strconv.FormatUint(lamportClock, 10))

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("[Replicator] Failed to reach peer %s: %v\n", peerURL, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("[Replicator] Successfully replicated %s to %s\n", fileName, peerURL)
			} else {
				fmt.Printf("[Replicator] Peer %s refused replication: status %d\n", peerURL, resp.StatusCode)
			}
		}(peer)
	}
}

// ReplicateDeleteToPeers sends a DELETE request to all configured backup nodes
func ReplicateDeleteToPeers(fileName string, lamportClock uint64) {
	cfg := config.Load()

	if len(cfg.Peers) == 0 {
		fmt.Println("[Replicator] No peers configured for deletion broadcast.")
		return
	}

	for _, peer := range cfg.Peers {
		// Construct the internal delete URL
		deleteURL := fmt.Sprintf("%s/internal/delete/%s", peer, fileName)

		req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
		if err != nil {
			fmt.Printf("[Replicator] Failed to create delete request for %s: %v\n", peer, err)
			continue
		}

		req.Header.Set("X-Lamport-Clock", strconv.FormatUint(lamportClock, 10))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[Replicator] Peer %s unreachable for deletion: %v\n", peer, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf("[Replicator] Successfully deleted replica %s from %s\n", fileName, peer)
		} else {
			fmt.Printf("[Replicator] Peer %s failed to delete: status %d\n", peer, resp.StatusCode)
		}
	}
}
