package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	clock "github.com/DS_node/Clock"
	"github.com/DS_node/config"
	"github.com/DS_node/election"
	"github.com/DS_node/models"
	"github.com/DS_node/repositories"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func leaderBaseURL() string {
	leaderID := election.CurrentLeaderID()
	if leaderID == "" {
		return ""
	}
	port := strings.TrimPrefix(leaderID, "node-")
	if port == "" || port == leaderID {
		return ""
	}
	return "http://localhost:" + port
}

func replicateUserToPeers(user models.User) {
	cfg := config.Load()
	if len(cfg.Peers) == 0 {
		return
	}

	payload, err := json.Marshal(user)
	if err != nil {
		fmt.Printf("[UserReplication] Failed to encode payload: %v\n", err)
		return
	}

	for _, peer := range cfg.Peers {
		go func(peerURL string) {
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Post(peerURL+"/internal/users", "application/json", bytes.NewReader(payload))
			if err != nil {
				fmt.Printf("[UserReplication] Failed to reach peer %s: %v\n", peerURL, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
				fmt.Printf("[UserReplication] Peer %s returned status %d\n", peerURL, resp.StatusCode)
			}
		}(peer)
	}
}

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

	if !election.IsCurrentNodeLeader() {
		if c.GetHeader("X-Forwarded-To-Leader") == "true" {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Unable to route request to leader"})
			return
		}

		leaderURL := leaderBaseURL()
		if leaderURL == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Leader not available"})
			return
		}

		payload, err := json.Marshal(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal request"})
			return
		}

		req, err := http.NewRequest(http.MethodPost, leaderURL+"/createUser", bytes.NewReader(payload))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create leader request"})
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-To-Leader", "true")
		req.Header.Set("X-Lamport-Clock", strconv.FormatUint(clockValue, 10))

		client := &http.Client{Timeout: 6 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to contact leader"})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	if user.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	_, err := repositories.GetUserByEmail(user.Email)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User Already Exists"})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := repositories.CreateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	replicateUserToPeers(user)

	c.JSON(http.StatusCreated, user)
}

func InternalCreateUser(c *gin.Context) {
	// Lamport Clock: Sync with sender's clock if provided (peer communication)
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

	fmt.Printf("[LamportClock] Internal create user event received. Clock advanced to: %d\n", clockValue)

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	_, err := repositories.GetUserByEmail(user.Email)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User Already Exists"})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := repositories.CreateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}
