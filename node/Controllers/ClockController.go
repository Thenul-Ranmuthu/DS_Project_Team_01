package controllers

import (
	"net/http"

	clock "github.com/DS_node/Clock"
	"github.com/gin-gonic/gin"
)

func GetClock(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"lamport_clock": clock.Node.Value(),
	})
}