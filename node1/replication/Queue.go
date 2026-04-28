package replication

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/models"
)

// AddToQueue records a failed replication for later retry
func AddToQueue(repType models.ReplicationType, peerURL string, payload string, filePath string) {
	pending := models.PendingReplication{
		Type:          repType,
		TargetPeer:    peerURL,
		Payload:       payload,
		FilePath:      filePath,
		Attempts:      0,
		LastAttemptAt: time.Now().Unix(),
		Status:        "pending",
	}

	if err := initializers.DB.Create(&pending).Error; err != nil {
		fmt.Printf("[ReplicationQueue] Failed to save pending replication: %v\n", err)
	} else {
		fmt.Printf("[ReplicationQueue] Added %s for %s to retry queue\n", repType, peerURL)
	}
}

// StartRetryWorker periodically checks the DB for failed replications and retries them
func StartRetryWorker() {
	fmt.Println("[ReplicationQueue] Starting background retry worker...")
	ticker := time.NewTicker(30 * time.Second) // Retry every 30 seconds
	go func() {
		for range ticker.C {
			processQueue()
		}
	}()
}

func processQueue() {
	var pendings []models.PendingReplication
	// Fetch pending replications that haven't exceeded max attempts (e.g., 10)
	err := initializers.DB.Where("status = ? AND attempts < ?", "pending", 10).Find(&pendings).Error
	if err != nil {
		return
	}

	if len(pendings) > 0 {
		fmt.Printf("[ReplicationQueue] Processing %d pending replications...\n", len(pendings))
	}

	for _, p := range pendings {
		success := false
		var err error

		switch p.Type {
		case models.ReplicateFileUpload:
			success, err = retryFileUpload(p)
		case models.ReplicateFileDelete:
			success, err = retryFileDelete(p)
		case models.ReplicateUserCreate:
			success, err = retryUserCreate(p)
		}

		p.Attempts++
		p.LastAttemptAt = time.Now().Unix()

		if success {
			p.Status = "completed"
			initializers.DB.Delete(&p) // Remove successful ones or mark completed
			fmt.Printf("[ReplicationQueue] Successfully retried %s for %s\n", p.Type, p.TargetPeer)
		} else {
			initializers.DB.Save(&p)
			fmt.Printf("[ReplicationQueue] Retry failed for %s to %s: %v (Attempt %d)\n", p.Type, p.TargetPeer, err, p.Attempts)
		}
	}
}

func retryFileUpload(p models.PendingReplication) (bool, error) {
	// Re-construct the multipart request
	file, err := os.Open(p.FilePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	var metadata map[string]interface{}
	json.Unmarshal([]byte(p.Payload), &metadata)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fileName := metadata["file_name"].(string)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return false, err
	}
	io.Copy(part, file)

	writer.WriteField("user_id", fmt.Sprintf("%v", metadata["user_id"]))
	writer.WriteField("original_name", metadata["original_name"].(string))
	writer.WriteField("mime_type", metadata["mime_type"].(string))
	writer.WriteField("file_size", fmt.Sprintf("%v", metadata["file_size"]))
	writer.Close()

	targetURL := fmt.Sprintf("%s/internal/replicate", p.TargetPeer)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(targetURL, writer.FormDataContentType(), body)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func retryFileDelete(p models.PendingReplication) (bool, error) {
	deleteURL := fmt.Sprintf("%s/internal/delete/%s", p.TargetPeer, p.Payload)
	req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
	if err != nil {
		return false, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func retryUserCreate(p models.PendingReplication) (bool, error) {
	targetURL := p.TargetPeer + "/internal/users"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(targetURL, "application/json", bytes.NewReader([]byte(p.Payload)))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// 201 Created or 409 Conflict (already exists) both count as success for retry
	return resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict, nil
}
