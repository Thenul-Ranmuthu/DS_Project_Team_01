package controllers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	clock "github.com/DS_node/Clock"
	"github.com/DS_node/Initializers"
	"github.com/DS_node/models"
	"github.com/DS_node/repositories"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
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
			clockValue = clock.Node.Sync(senderClock)
		} else {
			clockValue = clock.Node.Tick()
		}
	} else {
		clockValue = clock.Node.Tick()
	}

	fmt.Printf("[LamportClock] Upload event received. Clock advanced to: %d\n", clockValue)

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
	bucketName := initializers.GetBucketName()
	nodeID := os.Getenv("NODE_ID")

	var savedRecords []models.UploadedFile

	for _, fileHeader := range files {
		ext := filepath.Ext(fileHeader.Filename)
		objectName := fmt.Sprintf("%d%s", clock.NTP.Now().UnixNano(), ext)

		// WAL: log PENDING before any mutation
		walPayload := map[string]any{
			"original_name": fileHeader.Filename,
			"storage_key":   objectName,
			"user_id":       usr.ID,
			"file_size":     fileHeader.Size,
		}
		walEntry, walErr := repositories.CreateWALEntry(models.WALOpUpload, walPayload, nodeID)
		if walErr != nil {
			fmt.Printf("[WAL] Failed to create WAL entry for UPLOAD (%s): %v\n", fileHeader.Filename, walErr)
		}

		file, err := fileHeader.Open()
		if err != nil {
			if walEntry != nil {
				repositories.MarkWALFailed(walEntry.LogID)
				fmt.Printf("[WAL] UPLOAD failed (open) — WAL entry %d marked FAILED\n", walEntry.LogID)
			}
			continue
		}
		defer file.Close()

		_, err = initializers.MinioClient.PutObject(c.Request.Context(), bucketName, objectName, file, fileHeader.Size, minio.PutObjectOptions{
			ContentType: fileHeader.Header.Get("Content-Type"),
		})
		if err != nil {
			fmt.Println("MinIO Upload Error:", err)
			if walEntry != nil {
				repositories.MarkWALFailed(walEntry.LogID)
				fmt.Printf("[WAL] UPLOAD failed (minio) — WAL entry %d marked FAILED\n", walEntry.LogID)
			}
			continue
		}

		mimeType, _ := detectMIME(fileHeader)

		record := models.UploadedFile{
			OriginalName: fileHeader.Filename,
			StorageKey:   objectName,
			MimeType:     mimeType,
			FileSize:     fileHeader.Size,
			UserID:       usr.ID,
		}

		if err := repositories.CreateFile(&record); err == nil {
			if walEntry != nil {
				repositories.MarkWALCompleted(walEntry.LogID)
				fmt.Printf("[WAL] UPLOAD committed (%s) — WAL entry %d marked COMPLETED\n", fileHeader.Filename, walEntry.LogID)
			}
			savedRecords = append(savedRecords, record)
		} else {
			if walEntry != nil {
				repositories.MarkWALFailed(walEntry.LogID)
				fmt.Printf("[WAL] UPLOAD failed (db) — WAL entry %d marked FAILED\n", walEntry.LogID)
			}
		}
	}

	// Return the current clock value so the caller can synchronise their own Lamport clock.
	c.JSON(http.StatusOK, gin.H{
		"message":       fmt.Sprintf("%d file(s) uploaded", len(savedRecords)),
		"files":         savedRecords,
		"lamport_clock": clockValue,
	})
}