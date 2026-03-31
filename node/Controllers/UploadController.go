package controllers

import (
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"

	clock "github.com/DS_node/Clock"
	"github.com/DS_node/models"
	"github.com/DS_node/pkg/storage"
	"github.com/DS_node/repositories"
	"github.com/gin-gonic/gin"
)

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
 
func UploadMultipleFiles(c *gin.Context) {
	// Lamport Clock: Sync with sender's clock if provided, otherwise just tick for this local upload event. 
	var clockValue uint64
	if senderClockStr := c.GetHeader("X-Lamport-Clock"); senderClockStr != "" {
		senderClock, err := strconv.ParseUint(senderClockStr, 10, 64)
		if err == nil {
			// Received a clock value from another node: sync before proceeding.
			clockValue = clock.Node.Sync(senderClock)
		} else {
			clockValue = clock.Node.Tick()
		}
	} else {
		// Local upload event: tick the clock.
		clockValue = clock.Node.Tick()
	}
 
	slog.Info("Upload event received", "lamport_clock", clockValue)
 
	// Resolve the uploading user by email.
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
	var savedRecords []models.UploadedFile
 
	for _, fileHeader := range files {
		ext := filepath.Ext(fileHeader.Filename)
		storedName := fmt.Sprintf("%d%s", clock.NTP.Now().UnixNano(), ext)
		savePath := filepath.Join("./uploads", storedName)
 
		if err := storage.FS.Save(fileHeader, savePath); err != nil {
			slog.Error("Failed to save file", "error", err, "path", savePath)
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
		}
	}
 
	// Return the current clock value so the caller (or another node)
	// can synchronise their own Lamport clock.
	c.JSON(http.StatusOK, gin.H{
		"message":       fmt.Sprintf("%d file(s) uploaded", len(savedRecords)),
		"files":         savedRecords,
		"lamport_clock": clockValue,
	})
}
 