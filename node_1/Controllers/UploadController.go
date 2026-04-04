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
	"github.com/DS_node/replication"
	"github.com/gin-gonic/gin"
)

// detectMIME helper to identify file types
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

// UploadMultipleFiles handles the primary write request (Leader side)
func UploadMultipleFiles(c *gin.Context) {
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
		// Generate the UNIQUE filename using NTP time
		storedName := fmt.Sprintf("%d%s", clock.NTP.Now().UnixNano(), ext)
		savePath := filepath.Join(uploadDir, storedName)

		if err := c.SaveUploadedFile(fileHeader, savePath); err != nil {
			continue
		}

		mimeType, _ := detectMIME(fileHeader)

		record := models.UploadedFile{
			OriginalName: fileHeader.Filename,
			FilePath:     savePath, // This path uses the storedName
			MimeType:     mimeType,
			FileSize:     fileHeader.Size,
			UserID:       usr.ID,
		}

		if err := repositories.CreateFile(&record); err == nil {
			savedRecords = append(savedRecords, record)

			// --- TRIGGER REPLICATION ---
			// FIX: We pass 'storedName' so the Follower saves it with the SAME name as the Leader
			fmt.Printf("[Replicator] Triggering replication for: %s\n", storedName)
			replication.ReplicateToPeers(savePath, storedName)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       fmt.Sprintf("%d file(s) uploaded", len(savedRecords)),
		"files":         savedRecords,
		"lamport_clock": clockValue,
	})
}

// InternalReplicate handles files sent from the Leader to the Backup (Follower side)
func InternalReplicate(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file received"})
		return
	}

	uploadDir := "./uploads"
	os.MkdirAll(uploadDir, os.ModePerm)
	
	// The follower saves the file using the EXACT filename provided by the Leader
	savePath := filepath.Join(uploadDir, file.Filename)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save replicated file"})
		return
	}

	fmt.Printf("[Replication] Successfully synchronized file: %s\n", file.Filename)
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// DeleteReplica handles deletion requests from the Leader (Follower side)
func DeleteReplica(c *gin.Context) {
	fileName := c.Param("filename")
	filePath := filepath.Join("./uploads", fileName)

	// Check if file exists before trying to delete
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("[Replication] Delete failed: %s not found locally\n", fileName)
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	if err := os.Remove(filePath); err != nil {
		fmt.Printf("[Replication] Error deleting file %s: %v\n", fileName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	fmt.Printf("[Replication] Successfully deleted replica: %s\n", fileName)
	c.JSON(http.StatusOK, gin.H{"message": "Replica deleted successfully"})
}