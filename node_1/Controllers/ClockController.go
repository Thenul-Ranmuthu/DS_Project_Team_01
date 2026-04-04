package controllers

import (
	"net/http"
	"time"

	clock "github.com/DS_node/Clock"
	"github.com/gin-gonic/gin"
)

func GetClock(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"lamport_clock": clock.Node.Value(),
	})
}

func GetTime(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "ntp_time":   clock.NTP.Now().UTC().Format(time.RFC3339Nano),
        "ntp_offset": clock.NTP.Offset().String(),
    })
}