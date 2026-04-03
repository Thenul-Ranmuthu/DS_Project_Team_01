package controllers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	clock "github.com/DS_node/Clock"
	"github.com/DS_node/models"
	"github.com/DS_node/repositories"
	"github.com/gin-gonic/gin"
)

func CreateUser(c *gin.Context) {

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

	fmt.Printf("[LamportClock] Creating user event. Clock advanced to: %d\n", clockValue)

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// WAL: write PENDING before mutating state
	nodeID := os.Getenv("NODE_ID")
	walEntry, walErr := repositories.CreateWALEntry(models.WALOpCreateUser, user, nodeID)
	if walErr != nil {
		fmt.Printf("[WAL] Failed to create WAL entry for CREATE_USER: %v\n", walErr)
	}

	if err := repositories.CreateUser(&user); err != nil {
		if walEntry != nil {
			repositories.MarkWALFailed(walEntry.LogID)
		}
		fmt.Printf("[WAL] CREATE_USER failed — WAL entry %d marked FAILED\n", walEntry.LogID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if walEntry != nil {
		repositories.MarkWALCompleted(walEntry.LogID)
		fmt.Printf("[WAL] CREATE_USER committed — WAL entry %d marked COMPLETED\n", walEntry.LogID)
	}

	c.JSON(http.StatusCreated, user)
}

