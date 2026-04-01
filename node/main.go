package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
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
}


// CORSMiddleware enables CORS for the frontend origin
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
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


	router.Run(":" + config.Load().Port)
}



