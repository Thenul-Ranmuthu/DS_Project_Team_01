package controllers

import (
	"fmt"
	"mime/multipart"
	"net/http"
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
			// Received a clock value from another node: sync before proceeding.
			clockValue = clock.Node.Sync(senderClock)
		} else {
			clockValue = clock.Node.Tick()
		}
	} else {
		// Local upload event: tick the clock.
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

	var savedRecords []models.UploadedFile

	for _, fileHeader := range files {
		ext := filepath.Ext(fileHeader.Filename)
		objectName := fmt.Sprintf("%d%s", clock.NTP.Now().UnixNano(), ext)

		file, err := fileHeader.Open()
		if err != nil {
			continue
		}
		defer file.Close()

		_, err = initializers.MinioClient.PutObject(c.Request.Context(), bucketName, objectName, file, fileHeader.Size, minio.PutObjectOptions{
			ContentType: fileHeader.Header.Get("Content-Type"),
		})
		if err != nil {
			fmt.Println("MinIO Upload Error:", err)
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
 