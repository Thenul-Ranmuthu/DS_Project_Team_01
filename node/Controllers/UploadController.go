package controllers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	clock "github.com/DS_node/Clock"
	"github.com/DS_node/models"
	"github.com/DS_node/repositories"
	"github.com/DS_node/replication" // Added for Member 2 tasks
	"github.com/gin-gonic/gin"
)

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
	uploadDir := "./uploads"
	os.MkdirAll(uploadDir, os.ModePerm)

	var savedRecords []models.UploadedFile

	for _, fileHeader := range files {
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
			fmt.Printf("[Replicator] Triggering replication for: %s\n", fileHeader.Filename)
			replication.ReplicateToPeers(savePath, fileHeader.Filename)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       fmt.Sprintf("%d file(s) uploaded", len(savedRecords)),
		"files":         savedRecords,
		"lamport_clock": clockValue,
	})
}

// InternalReplicate handles files sent from other nodes (Backup Node)
// This is called by ReplicateToPeers from the Leader node.
func InternalReplicate(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file received for replication"})
		return
	}

	uploadDir := "./uploads"
	os.MkdirAll(uploadDir, os.ModePerm)

	// We use the original filename to ensure consistency across the cluster
	savePath := filepath.Join(uploadDir, file.Filename)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		fmt.Printf("[Replication] Failed to save replicated file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save replicated file"})
		return
	}

	fmt.Printf("[Replication] Successfully synchronized file: %s\n", file.Filename)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "File replicated successfully",
	})
}