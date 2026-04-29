package replication

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/election"
	"github.com/DS_node/models"
)

func leaderBaseURL() string {
	leaderID := election.CurrentLeaderID()
	if leaderID == "" {
		return ""
	}
	// Extract port from node ID (assuming node-5051 format)
	port := strings.TrimPrefix(leaderID, "node-")
	if port == "" || port == leaderID {
		return ""
	}
	return "http://localhost:" + port
}

// TriggerRecovery starts the automated catch-up process
func TriggerRecovery() {
	fmt.Println("[Recovery] Starting automated catch-up...")
	
	// Wait a bit for ZK to settle
	time.Sleep(5 * time.Second)

	leaderURL := leaderBaseURL()
	if leaderURL == "" {
		fmt.Println("[Recovery] Leader not found. Skipping catch-up.")
		return
	}

	// 1. Sync Users
	syncUsers(leaderURL)

	// 2. Sync Files
	syncFiles(leaderURL)

	fmt.Println("[Recovery] Catch-up completed.")
}

func syncUsers(leaderURL string) {
	fmt.Println("[Recovery] Syncing users from leader...")
	resp, err := http.Get(leaderURL + "/internal/users/all")
	if err != nil {
		fmt.Printf("[Recovery] Failed to fetch users: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var leaderUsers []models.User
	json.NewDecoder(resp.Body).Decode(&leaderUsers)

	for _, u := range leaderUsers {
		var localUser models.User
		if err := initializers.DB.Where("email = ?", u.Email).First(&localUser).Error; err != nil {
			// User missing, create it
			fmt.Printf("[Recovery] Adding missing user: %s\n", u.Email)
			u.ID = 0 // Reset ID for local DB
			initializers.DB.Create(&u)
		}
	}
}

func syncFiles(leaderURL string) {
	fmt.Println("[Recovery] Syncing files from leader...")
	resp, err := http.Get(leaderURL + "/internal/files/all")
	if err != nil {
		fmt.Printf("[Recovery] Failed to fetch files: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var leaderFiles []models.UploadedFile
	json.NewDecoder(resp.Body).Decode(&leaderFiles)

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	for _, f := range leaderFiles {
		fileName := filepath.Base(f.FilePath)
		localPath := filepath.Join(uploadDir, fileName)

		var localFile models.UploadedFile
		dbErr := initializers.DB.Where("file_path LIKE ?", "%"+fileName).First(&localFile).Error

		if dbErr != nil || !fileExists(localPath) {
			fmt.Printf("[Recovery] Catching up missing file: %s\n", fileName)
			
			// Download file content
			if err := downloadFile(leaderURL+"/internal/files/download/"+fileName, localPath); err != nil {
				fmt.Printf("[Recovery] Failed to download %s: %v\n", fileName, err)
				continue
			}

			// Upsert DB record
			if dbErr != nil {
				f.ID = 0 // Reset ID
				f.FilePath = localPath
				initializers.DB.Create(&f)
			}
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func downloadFile(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
