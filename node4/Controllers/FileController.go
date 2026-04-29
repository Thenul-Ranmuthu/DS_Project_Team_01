package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	// Ensure these match your folder structure exactly
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

	// Lamport Clock logic
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

	fmt.Printf("[LamportClock] Delete event received. Clock: %d\n", clockValue)

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	// 1. Get record from DB
	file, err := repositories.GetFileByID(uint(fileID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// 2. Local Delete
	if err := os.Remove(file.FilePath); err != nil {
		fmt.Printf("[Warning] Local disk delete failed: %v\n", err)
	}

	// 3. --- BROADCAST DELETE ---
	// Extract the actual filename (e.g. 1712248593.png)
	fileName := filepath.Base(file.FilePath)
	fmt.Printf("[Replicator] Telling peers to delete: %s\n", fileName)

	// Calling the replication package
	replication.ReplicateDeleteToPeers(fileName, clockValue)

	// 4. Delete from DB
	if err := repositories.DeleteFile(uint(fileID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete DB record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "File deleted from cluster successfully",
		"lamport_clock": clockValue,
	})
}
