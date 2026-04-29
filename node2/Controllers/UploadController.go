package controllers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	clock "github.com/DS_node/Clock"
	"github.com/DS_node/config"
	"github.com/DS_node/election"
	"github.com/DS_node/models"
	"github.com/DS_node/replication" // Added for Member 2 tasks
	"github.com/DS_node/repositories"
	"github.com/gin-gonic/gin"
)

func getUploadDir() string {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		return "./uploads"
	}
	return uploadDir
}

func isBlockedExtension(fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	return ext == ".md"
}

// detectMIME reads the first 512 bytes of a file to determine its actual content type
func detectMIME(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 512)
	_, err = file.Read(buf)
	if err != nil {
		return "", err
	}

	return http.DetectContentType(buf), nil
}

// UploadMultipleFiles handles the primary upload from a client and triggers replication
func UploadMultipleFiles(c *gin.Context) {
	// Lamport Clock: Sync with sender's clock if provided
	var clockValue uint64
	if senderClockStr := c.GetHeader("X-Lamport-Clock"); senderClockStr != "" {
		senderClock, err := strconv.ParseUint(senderClockStr, 10, 64)
		if err == nil {
			clockValue = clock.Node.Sync(senderClock)
		} else {
			clockValue = clock.Node.Tick()
		}
	} else {
		clockValue = clock.Node.Tick()
	}

	fmt.Printf("[LamportClock] Upload event received. Clock advanced to: %d\n", clockValue)
 
	// Partition Check: Reject writes if we are a follower and cannot reach the leader
	if !election.IsCurrentNodeLeader() && !election.IsLeaderReachable() {
		fmt.Printf("[Partition] Rejecting upload: Leader unreachable\n")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Network partition detected: leader is unreachable. System is in read-only mode.",
		})
		return
	}

	// Resolve the uploading user by email
	usr, errorUser := repositories.GetUserByEmail(c.Param("email"))
	if errorUser != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Email!!"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid multipart form"})
		return
	}

	files := form.File["files"]
	uploadDir := getUploadDir()
	os.MkdirAll(uploadDir, os.ModePerm)

	var savedRecords []models.UploadedFile
	var quorumFailed bool

	for _, fileHeader := range files {
		if isBlockedExtension(fileHeader.Filename) {
			fmt.Printf("[Upload] Skipping blocked file type: %s\n", fileHeader.Filename)
			continue
		}

		ext := filepath.Ext(fileHeader.Filename)
		// Using NTP time for unique storage naming
		storedName := fmt.Sprintf("%d%s", clock.NTP.Now().UnixNano(), ext)
		savePath := filepath.Join(uploadDir, storedName)

		if err := c.SaveUploadedFile(fileHeader, savePath); err != nil {
			continue
		}

		mimeType, _ := detectMIME(fileHeader)

		record := models.UploadedFile{
			OriginalName: fileHeader.Filename,
			FilePath:     savePath,
			MimeType:     mimeType,
			FileSize:     fileHeader.Size,
			UserID:       usr.ID,
		}

		if err := repositories.CreateFile(&record); err == nil {
			savedRecords = append(savedRecords, record)

			// --- MEMBER 2: REPLICATION TRIGGER ---
			// After saving locally on the Leader, push this file to the Backup peers
			fmt.Printf("[Replicator] Triggering replication for: %s\n", storedName)
			repResults := replication.ReplicateToPeers(savePath, storedName, usr.ID, fileHeader.Filename, mimeType, fileHeader.Size)
			
			// Quorum check:
			successCount := 1 // Start with 1 for this node
			for _, res := range repResults {
				if res.Success {
					successCount++
				}
			}

			cfg := config.Load()
			totalNodes := len(cfg.Peers) + 1
			quorum := (totalNodes / 2) + 1

			if successCount < quorum {
				fmt.Printf("[Upload] Quorum not met for %s: %d/%d nodes succeeded\n", storedName, successCount, totalNodes)
				quorumFailed = true
			} else {
				fmt.Printf("[Upload] Quorum met for %s: %d/%d nodes succeeded\n", storedName, successCount, totalNodes)
			}
		}
	}

	// Calculate overall success
	status := http.StatusOK
	message := fmt.Sprintf("%d file(s) uploaded", len(savedRecords))
	if quorumFailed {
		status = http.StatusAccepted
		message += " (Warning: Replication quorum not met for some files. Retries queued.)"
	}

	c.JSON(status, gin.H{
		"message":       message,
		"files":         savedRecords,
		"lamport_clock": clockValue,
	})
}

// InternalReplicate handles files sent from other nodes (Backup Node)
// This is called by ReplicateToPeers from the Leader node.
func InternalReplicate(c *gin.Context) {
	// Lamport Clock: Sync with sender's clock if provided (peer communication)
	var clockValue uint64
	if senderClockStr := c.GetHeader("X-Lamport-Clock"); senderClockStr != "" {
		senderClock, err := strconv.ParseUint(senderClockStr, 10, 64)
		if err == nil {
			clockValue = clock.Node.Sync(senderClock)
		} else {
			clockValue = clock.Node.Tick()
		}
	} else {
		clockValue = clock.Node.Tick()
	}

	fmt.Printf("[LamportClock] Internal replicate event received. Clock advanced to: %d\n", clockValue)

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file received for replication"})
		return
	}

	if isBlockedExtension(file.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Blocked file type"})
		return
	}

	uploadDir := getUploadDir()
	os.MkdirAll(uploadDir, os.ModePerm)

	// We use the original filename to ensure consistency across the cluster
	savePath := filepath.Join(uploadDir, file.Filename)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		fmt.Printf("[Replication] Failed to save replicated file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save replicated file"})
		return
	}

	// Extract metadata from form fields
	userIDStr := c.PostForm("user_id")
	originalName := c.PostForm("original_name")
	mimeType := c.PostForm("mime_type")
	fileSizeStr := c.PostForm("file_size")

	userID, _ := strconv.ParseUint(userIDStr, 10, 64)
	fileSize, _ := strconv.ParseInt(fileSizeStr, 10, 64)

	// Create database record for the replicated file
	if userID > 0 {
		record := models.UploadedFile{
			OriginalName: originalName,
			FilePath:     savePath,
			MimeType:     mimeType,
			FileSize:     fileSize,
			UserID:       uint(userID),
		}
		if err := repositories.CreateFile(&record); err != nil {
			fmt.Printf("[Replication] Failed to create database record: %v\n", err)
			// Don't fail the request, but log it
		} else {
			fmt.Printf("[Replication] Database record created for file: %s\n", file.Filename)
		}
	}

	fmt.Printf("[Replication] Successfully synchronized file: %s\n", file.Filename)
	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"message":       "File replicated successfully",
		"lamport_clock": clockValue,
	})
}

// DeleteReplica handles deletion requests sent from peer nodes.
func DeleteReplica(c *gin.Context) {
	// Lamport Clock: Sync with sender's clock if provided (peer communication)
	var clockValue uint64
	if senderClockStr := c.GetHeader("X-Lamport-Clock"); senderClockStr != "" {
		senderClock, err := strconv.ParseUint(senderClockStr, 10, 64)
		if err == nil {
			clockValue = clock.Node.Sync(senderClock)
		} else {
			clockValue = clock.Node.Tick()
		}
	} else {
		clockValue = clock.Node.Tick()
	}

	fmt.Printf("[LamportClock] Delete replica event received. Clock advanced to: %d\n", clockValue)

	fileName := c.Param("filename")
	filePath := filepath.Join(getUploadDir(), fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("[Replication] Delete skipped: %s not found locally\n", fileName)
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	if err := os.Remove(filePath); err != nil {
		fmt.Printf("[Replication] Error deleting file %s: %v\n", fileName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	// Delete database record by filepath
	if err := repositories.DeleteFileByPath(filePath); err != nil {
		fmt.Printf("[Replication] Warning: Failed to delete database record for %s: %v\n", fileName, err)
		// Don't fail the request, but log it
	} else {
		fmt.Printf("[Replication] Database record deleted for file: %s\n", fileName)
	}

	fmt.Printf("[Replication] Successfully deleted replica: %s\n", fileName)
	c.JSON(http.StatusOK, gin.H{
		"message":       "Replica deleted successfully",
		"lamport_clock": clockValue,
	})
}
