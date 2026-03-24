package controllers

import (
	"fmt"
	"net/http"
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
			// Received a clock value from another node: sync before proceeding.
			clockValue = clock.Node.Sync(senderClock)
		} else {
			clockValue = clock.Node.Tick()
		}
	} else {
		// Local upload event: tick the clock.
		clockValue = clock.Node.Tick()
	}
 
	fmt.Printf("[LamportClock] Creating user event. Clock advanced to: %d\n", clockValue)

	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := repositories.CreateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}
