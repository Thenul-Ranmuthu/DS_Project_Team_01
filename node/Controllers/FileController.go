package controllers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	clock "github.com/DS_node/Clock"
	"github.com/DS_node/Initializers"
	"github.com/DS_node/models"
	"github.com/DS_node/repositories"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

func GetUserFiles(c *gin.Context) {

	usr, errorUser := repositories.GetUserByEmail(c.Param("email"))
	if errorUser != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Email!!"})
		return
	}

	files, err := repositories.GetFilesByUser(usr.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch files"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

func GetFileByID(c *gin.Context) {
	fileIDStr := c.Param("id")
	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	file, err := repositories.GetFileByID(uint(fileID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file": file})
}

func DeleteFile(c *gin.Context) {
	fileIDStr := c.Param("id")

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

	fmt.Printf("[LamportClock] Delete event received. Clock advanced to: %d\n", clockValue)

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	// Fetch the file record first to get the file key
	file, err := repositories.GetFileByID(uint(fileID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// WAL: log PENDING before any mutation
	nodeID := os.Getenv("NODE_ID")
	walPayload := map[string]any{
		"file_id":     fileID,
		"storage_key": file.StorageKey,
	}
	walEntry, walErr := repositories.CreateWALEntry(models.WALOpDelete, walPayload, nodeID)
	if walErr != nil {
		fmt.Printf("[WAL] Failed to create WAL entry for DELETE (file %d): %v\n", fileID, walErr)
	}

	bucketName := initializers.GetBucketName()
	if err := initializers.MinioClient.RemoveObject(c.Request.Context(), bucketName, file.StorageKey, minio.RemoveObjectOptions{}); err != nil {
		fmt.Println("Warning: could not delete file from MinIO:", err)
		// Non-fatal: proceed with DB delete even if MinIO object is already gone
	}

	// Delete the DB record
	if err := repositories.DeleteFile(uint(fileID)); err != nil {
		if walEntry != nil {
			repositories.MarkWALFailed(walEntry.LogID)
			fmt.Printf("[WAL] DELETE failed (db) — WAL entry %d marked FAILED\n", walEntry.LogID)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	if walEntry != nil {
		repositories.MarkWALCompleted(walEntry.LogID)
		fmt.Printf("[WAL] DELETE committed (file %d, key: %s) — WAL entry %d marked COMPLETED\n", fileID, file.StorageKey, walEntry.LogID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

