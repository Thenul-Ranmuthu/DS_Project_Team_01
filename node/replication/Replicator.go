package replication

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/DS_node/config"
)

// ReplicateToPeers sends the uploaded file to all other nodes in the cluster
func ReplicateToPeers(filePath string, fileName string) {
	cfg := config.Load()

	for _, peer := range cfg.Peers {
		// We use a goroutine (go func) so the user doesn't have to wait 
		// for the backup to finish before getting a "Success" message.
		go func(peerURL string) {
			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("[Replicator] Error opening local file: %v\n", err)
				return
			}
			defer file.Close()

			// Prepare the form data to send to the peer
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("file", fileName)
			io.Copy(part, file)
			writer.Close()

			// We will create this "internal/replicate" endpoint in the next step
			targetURL := fmt.Sprintf("%s/internal/replicate", peerURL)
			
			resp, err := http.Post(targetURL, writer.FormDataContentType(), body)
			if err != nil {
				fmt.Printf("[Replicator] Failed to reach peer %s: %v\n", peerURL, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("[Replicator] Successfully replicated %s to %s\n", fileName, peerURL)
			}
		}(peer)
	}
}