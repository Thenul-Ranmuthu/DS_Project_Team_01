package controllers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/DS_node/models"
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

// func UploadSingleFile(c *gin.Context) {
// 	userID := c.MustGet("userID").(uint)

// 	fileHeader, err := c.FormFile("file")
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
// 		return
// 	}

// 	// Validate file size (max 10MB)
// 	if fileHeader.Size > 10<<20 {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 10MB)"})
// 		return
// 	}

// 	// Detect MIME type
// 	mimeType, err := detectMIME(fileHeader)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not detect file type"})
// 		return
// 	}

// 	// Validate allowed types
// 	allowed := map[string]bool{
// 		"image/jpeg":      true,
// 		"image/png":       true,
// 		"application/pdf": true,
// 	}
// 	if !allowed[mimeType] {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed"})
// 		return
// 	}

// 	// Build save path
// 	uploadDir := "./uploads"
// 	os.MkdirAll(uploadDir, os.ModePerm)

// 	ext := filepath.Ext(fileHeader.Filename)
// 	storedName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
// 	savePath := filepath.Join(uploadDir, storedName)

// 	if err := c.SaveUploadedFile(fileHeader, savePath); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
// 		return
// 	}

// 	record := models.UploadedFile{
// 		OriginalName: fileHeader.Filename,
// 		FilePath:     savePath,
// 		MimeType:     mimeType,
// 		FileSize:     fileHeader.Size,
// 		UserID:       userID,
// 	}

// 	if err := repositories.CreateFile(&record); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save record"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "File uploaded successfully",
// 		"file":    record,
// 	})
// }

func UploadMultipleFiles(c *gin.Context) {
	// userID := c.MustGet("userID").(uint)

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
		storedName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
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
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%d file(s) uploaded", len(savedRecords)),
		"files":   savedRecords,
	})
}
