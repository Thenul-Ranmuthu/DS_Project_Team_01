package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath" // Added for filepath.Base
	"strconv"

	clock "github.com/DS_node/Clock"
	"github.com/DS_node/replication"
	"github.com/DS_node/repositories"
	"github.com/gin-gonic/gin"
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

	// 1. Fetch the file record first to get the file path
	file, err := repositories.GetFileByID(uint(fileID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// 2. Delete the physical file from the Leader's disk
	if err := os.Remove(file.FilePath); err != nil {
		fmt.Println("Warning: could not delete file from disk:", err)
	}

	// 3. --- TRIGGER REPLICATION DELETE ---
	// This sends the delete command to all Followers
	fmt.Printf("[Replicator] Broadcasting delete for: %s\n", file.OriginalName)
	replication.ReplicateDeleteToPeers(filepath.Base(file.FilePath))

	// 4. Delete the DB record
	if err := repositories.DeleteFile(uint(fileID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file from database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "File deleted from cluster successfully",
		"lamport_clock": clockValue,
	})
}