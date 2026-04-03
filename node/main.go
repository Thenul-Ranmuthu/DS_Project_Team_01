package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	clock "github.com/DS_node/Clock"
	controllers "github.com/DS_node/Controllers"
	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/config"
	"github.com/DS_node/election"
	"github.com/DS_node/migrate"
	"github.com/DS_node/models"
	"github.com/DS_node/repositories"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

var (
	em         *election.ElectionManager
	AppMetrics struct {
		sync.RWMutex
		DBLatencyMs    int64
		UploadFailures int64
		LeaderChanges  int64
		DBReconnects   int64
	}
)

func init() {

	initializers.LoadEnvVaribles()
	migrate.MigrateDB()
	initializers.InitStorage()

	// Initial sync at startup
	if err := clock.NTP.Sync("pool.ntp.org"); err != nil {
		log.Printf("[NTPClock] Initial sync failed: %v", err)
	} else {
		log.Println("[NTPClock] Initial sync successful")
	}


	// Re-sync every 10 minutes in the background
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			if err := clock.NTP.Sync("pool.ntp.org"); err != nil {
				log.Printf("[NTPClock] Re-sync failed: %v", err)
			} else {
				log.Println("[NTPClock] Background sync successful")
			}
		}
	}()


	cfg := config.Load()
	zkServers := cfg.ZKServers
	nodeID := cfg.NodeID
	if nodeID == "" {
		log.Fatal("NODE_ID environment variable is required")
	}

	var err error
	em, err = election.NewElectionManager(zkServers, nodeID)
	if err != nil {
		log.Fatalf("Election manager init failed: %v", err)
	}

	// Set callbacks to integrate with your replication/clock packages

	em.SetOnBecomeLeader(func() {
		em.LogEvent("Confirmed: Node is now LEADER — enabling write access")
		initializers.DB.Create(&models.ElectionEvent{
			NodeID:    nodeID,
			EventType: "became_leader",
		})
	})


	em.SetOnLeaderChanged(func(newLeaderID string) {
		AppMetrics.Lock()
		AppMetrics.LeaderChanges++
		changes := AppMetrics.LeaderChanges
		AppMetrics.Unlock()
		
		em.LogEvent(fmt.Sprintf("Leader changed to %s. Total changes: %d", newLeaderID, changes))
		if changes > 5 {
			em.LogEvent("[ALERT] Leader changes are happening too often! Possible network flap or ZK overload.")
		}
	})


	// Background metrics loop for DB latency & reconnect tracking
	go func() {
		dbStatusOk := true
		for {
			time.Sleep(5 * time.Second)
			sqlDB, err := initializers.DB.DB()
			if err != nil {
				continue
			}

			start := time.Now()
			pingErr := sqlDB.Ping()
			latency := time.Since(start).Milliseconds()

			AppMetrics.Lock()
			AppMetrics.DBLatencyMs = latency
			if pingErr != nil {
				if dbStatusOk {
					dbStatusOk = false
					AppMetrics.DBReconnects++
					em.LogEvent(fmt.Sprintf("[ALERT] DB connection lost! Total reconnects: %d", AppMetrics.DBReconnects))
				}
			} else {
				if !dbStatusOk {
					em.LogEvent("DB connection recovered.")
					dbStatusOk = true
				}
			}

			AppMetrics.Unlock()
		}
	}()

	go func() {
		if err := em.Start(); err != nil {
			log.Fatalf("Election failed: %v", err)
		}
	}()

	// Follower WAL Sync: periodically pull COMPLETED WAL entries from the leader
	// and replay metadata changes locally. MinIO handles file durability independently.
	go func() {
		var lastSeenLogID uint64 = 0
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			if em.IsLeader() {
				continue // Leaders write WAL; they don't pull from themselves
			}
			leaderID := em.LeaderID()
			if leaderID == "" {
				continue
			}
			// Derive leader port from leaderID (node_1 -> 8000, node_2 -> 8001, ...)
			leaderPort := 8000
			if strings.HasPrefix(leaderID, "node_") {
				if idx, err := strconv.Atoi(strings.TrimPrefix(leaderID, "node_")); err == nil {
					leaderPort = 8000 + (idx - 1)
				}
			}
			syncURL := fmt.Sprintf("http://localhost:%d/replication/sync?after=%d", leaderPort, lastSeenLogID)
			resp, err := http.Get(syncURL)
			if err != nil {
				log.Printf("[WAL-Sync] Could not reach leader at port %d: %v", leaderPort, err)
				continue
			}
			var entries []models.WriteAheadLog
			if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
				resp.Body.Close()
				log.Printf("[WAL-Sync] Failed to decode sync response: %v", err)
				continue
			}
			resp.Body.Close()

			for _, entry := range entries {
				switch entry.Operation {
				case models.WALOpUpload:
					var payload struct {
						OriginalName string `json:"original_name"`
						StorageKey   string `json:"storage_key"`
						MimeType     string `json:"mime_type"`
						FileSize     int64  `json:"file_size"`
						UserID       uint   `json:"user_id"`
					}
					if err := json.Unmarshal([]byte(entry.Payload), &payload); err != nil {
						log.Printf("[WAL-Sync] Failed to parse UPLOAD payload (log %d): %v", entry.LogID, err)
						continue
					}
					record := &models.UploadedFile{
						OriginalName: payload.OriginalName,
						StorageKey:   payload.StorageKey,
						MimeType:     payload.MimeType,
						FileSize:     payload.FileSize,
						UserID:       payload.UserID,
					}
					if err := repositories.CreateFile(record); err != nil {
						log.Printf("[WAL-Sync] UPLOAD replay failed (log %d): %v", entry.LogID, err)
					} else {
						em.LogEvent(fmt.Sprintf("[WAL-Sync] Replayed UPLOAD log %d: %s", entry.LogID, payload.OriginalName))
					}
				case models.WALOpDelete:
					var payload struct {
						FileID uint `json:"file_id"`
					}
					if err := json.Unmarshal([]byte(entry.Payload), &payload); err != nil {
						log.Printf("[WAL-Sync] Failed to parse DELETE payload (log %d): %v", entry.LogID, err)
						continue
					}
					if err := repositories.DeleteFile(payload.FileID); err != nil {
						log.Printf("[WAL-Sync] DELETE replay failed (log %d): %v", entry.LogID, err)
					} else {
						em.LogEvent(fmt.Sprintf("[WAL-Sync] Replayed DELETE log %d: file_id=%d", entry.LogID, payload.FileID))
					}
				case models.WALOpCreateUser:
					var user models.User
					if err := json.Unmarshal([]byte(entry.Payload), &user); err != nil {
						log.Printf("[WAL-Sync] Failed to parse CREATE_USER payload (log %d): %v", entry.LogID, err)
						continue
					}
					if err := repositories.CreateUser(&user); err != nil {
						log.Printf("[WAL-Sync] CREATE_USER replay failed (log %d): %v", entry.LogID, err)
					} else {
						em.LogEvent(fmt.Sprintf("[WAL-Sync] Replayed CREATE_USER log %d: %s", entry.LogID, user.Email))
					}
				}
				if entry.LogID > lastSeenLogID {
					lastSeenLogID = entry.LogID
				}
			}
			if len(entries) > 0 {
				log.Printf("[WAL-Sync] Applied %d entries from leader (last log_id: %d)", len(entries), lastSeenLogID)
			}
		}
	}()
}


// CORSMiddleware enables CORS for the frontend origin
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, Idempotency-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// LeaderOnly middleware ensures only the leader processes write requests.
// If the node is a follower, it returns 423 with the current leader's info.
func LeaderOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !em.IsLeader() {
			leaderID := em.LeaderID()
			
			// Map node_1 -> 9000, node_2 -> 9001 (so frontend can -1000 to get 8000 series)
			port := 9000
			if strings.HasPrefix(leaderID, "node_") {
				idxStr := strings.TrimPrefix(leaderID, "node_")
				if idx, err := strconv.Atoi(idxStr); err == nil {
					port = 8000 + (idx - 1) + 1000
				}
			}

			c.JSON(423, gin.H{
				"error":  "Consensus error: write operation must be performed on leader.",
				"leader": fmt.Sprintf("localhost:%d", port),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}


func main() {
	router := gin.Default()
	router.Use(CORSMiddleware())

	router.POST("/createUser", LeaderOnly(), controllers.CreateUser)

	router.GET("/ping", controllers.PingEndPoint)

	// Observability & Monitoring
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "message": "process is alive"})
	})

	router.GET("/health/ready", func(c *gin.Context) {
		dbStatus := "ok"
		sqlDB, err := initializers.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "error"
		}
		
		zkStatus := "ok"
		if !em.IsConnected() {
			zkStatus = "error"
		}

		status := 200
		if dbStatus == "error" || zkStatus == "error" {
			status = 503
		}

		c.JSON(status, gin.H{
			"db": dbStatus,
			"zk": zkStatus,
			"ready": status == 200,
		})
	})

	router.GET("/metrics", func(c *gin.Context) {
		AppMetrics.RLock()
		defer AppMetrics.RUnlock()
		c.JSON(200, gin.H{
			"db_latency_ms":    AppMetrics.DBLatencyMs,
			"upload_failures":  AppMetrics.UploadFailures,
			"leader_changes":   AppMetrics.LeaderChanges,
			"db_reconnects":    AppMetrics.DBReconnects,
		})
	})

	router.POST("/upload/:email", LeaderOnly(), controllers.UploadMultipleFiles)
	router.GET("/users/files/:email", controllers.GetUserFiles)
	router.GET("/files/:id", controllers.GetFileByID)
	router.DELETE("/files/:id", LeaderOnly(), controllers.DeleteFile)

	// Lamport clock — lets other nodes (or a monitor) read this node's logical time
	router.GET("/clock", controllers.GetClock)

	router.POST("/upload", LeaderOnly(), func(c *gin.Context) {
		idempotencyKey := c.GetHeader("Idempotency-Key")
		if idempotencyKey != "" {
			var record models.IdempotencyRecord
			result := initializers.DB.First(&record, "key = ?", idempotencyKey)
			if result.RowsAffected > 0 {
				em.LogEvent(fmt.Sprintf("[Idempotency] Request with key %s already processed. Returning cached success.", idempotencyKey))
				c.JSON(record.StatusCode, gin.H{"message": "success", "cached": true})
				return
			}
		}

		fileHeader, err := c.FormFile("file")
		if err != nil {
			AppMetrics.Lock()
			AppMetrics.UploadFailures++
			AppMetrics.Unlock()
			c.String(400, "Get form err: %s", err.Error())
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			c.String(400, "Get file err: %s", err.Error())
			return
		}
		defer file.Close()

		bucketName := initializers.GetBucketName()
		objectName := fileHeader.Filename
		_, err = initializers.MinioClient.PutObject(c.Request.Context(), bucketName, objectName, file, fileHeader.Size, minio.PutObjectOptions{
			ContentType: fileHeader.Header.Get("Content-Type"),
		})

		if err != nil {
			AppMetrics.Lock()
			AppMetrics.UploadFailures++
			AppMetrics.Unlock()
			c.String(400, "Upload err: %s", err.Error())
			return
		}

		if idempotencyKey != "" {
			initializers.DB.Create(&models.IdempotencyRecord{
				Key:        idempotencyKey,
				StatusCode: 200,
				Body:       "success",
			})
			em.LogEvent(fmt.Sprintf("[Idempotency] Request with key %s processed successfully and cached.", idempotencyKey))
		}

		c.JSON(200, gin.H{"message": "success"})
	})

	router.GET("/files", func(c *gin.Context) {
		type FileInfo struct {
			Name    string `json:"name"`
			Size    int64  `json:"size"`
			ModTime string `json:"modTime"`
		}
		var files []FileInfo
		bucketName := initializers.GetBucketName()
		objectCh := initializers.MinioClient.ListObjects(c.Request.Context(), bucketName, minio.ListObjectsOptions{})
		
		for object := range objectCh {
			if object.Err != nil {
				continue
			}
			files = append(files, FileInfo{
				Name:    object.Key,
				Size:    object.Size,
				ModTime: object.LastModified.Format(time.RFC3339),
			})
		}
		if files == nil {
			files = make([]FileInfo, 0)
		}
		c.JSON(200, files)
	})

	router.GET("/download", func(c *gin.Context) {
		fileName := c.Query("file")
		bucketName := initializers.GetBucketName()
		
		reqParams := make(url.Values)
		presignedURL, err := initializers.MinioClient.PresignedGetObject(c.Request.Context(), bucketName, fileName, time.Hour*1, reqParams)
		if err != nil {
			c.String(500, "Download err: %s", err.Error())
			return
		}
		c.Redirect(http.StatusTemporaryRedirect, presignedURL.String())
	})

	router.GET("/status", func(c *gin.Context) {
		AppMetrics.RLock()
		term := AppMetrics.LeaderChanges
		AppMetrics.RUnlock()

		bucketName := initializers.GetBucketName()
		var applied int64
		for range initializers.MinioClient.ListObjects(c.Request.Context(), bucketName, minio.ListObjectsOptions{}) {
			applied++
		}
		peers := em.GetPeerCount()

		em.HandleStatus(c, applied, term, peers)
	})

	router.POST("/shutdown", em.HandleShutdown)

	router.GET("/election/status", func(c *gin.Context) {
		AppMetrics.RLock()
		term := AppMetrics.LeaderChanges
		AppMetrics.RUnlock()

		bucketName := initializers.GetBucketName()
		var applied int64
		for range initializers.MinioClient.ListObjects(c.Request.Context(), bucketName, minio.ListObjectsOptions{}) {
			applied++
		}
		peers := em.GetPeerCount()

		em.HandleStatus(c, applied, term, peers)
	})
	router.POST("/election/resign", em.HandleResign)

	// Replication Sync: leader serves COMPLETED WAL entries after a given log_id.
	// Followers poll this endpoint to replay missed metadata operations.
	router.GET("/replication/sync", func(c *gin.Context) {
		afterStr := c.DefaultQuery("after", "0")
		afterLogID, err := strconv.ParseUint(afterStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'after' parameter"})
			return
		}
		entries, err := repositories.GetCompletedWALAfter(afterLogID, 100)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch WAL entries"})
			return
		}
		if entries == nil {
			entries = []models.WriteAheadLog{}
		}
		nodeID := os.Getenv("NODE_ID")
		log.Printf("[WAL] Node %s served %d WAL entries (after log_id=%d)", nodeID, len(entries), afterLogID)
		c.JSON(http.StatusOK, entries)
	})

	router.Run(":" + config.Load().Port)
}



