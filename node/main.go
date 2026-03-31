package main

import (
	"log"
	"os"
	"time"

	clock "github.com/DS_node/Clock"
	controllers "github.com/DS_node/Controllers"
	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/config"
	"github.com/DS_node/election"
	"github.com/DS_node/migrate"
	"github.com/DS_node/models"
	"github.com/gin-gonic/gin"
)

var em *election.ElectionManager

func init() {
	initializers.LoadEnvVaribles()
	migrate.MigrateDB()

	// Initial sync at startup
	if err := clock.NTP.Sync("pool.ntp.org"); err != nil {
		log.Printf("[NTPClock] Initial sync failed: %v", err)
	}

	// Re-sync every 10 minutes in the background
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			if err := clock.NTP.Sync("pool.ntp.org"); err != nil {
				log.Printf("[NTPClock] Re-sync failed: %v", err)
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
		log.Println("This node is now leader — start accepting writes")
		initializers.DB.Create(&models.ElectionEvent{
			NodeID:    nodeID,
			EventType: "became_leader",
		})
	})

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

func main() {
	router := gin.Default()
	router.Use(CORSMiddleware())

	router.POST("/createUser", controllers.CreateUser)

	router.GET("/ping", controllers.PingEndPoint)

	router.POST("/upload/:email", controllers.UploadMultipleFiles)
	router.GET("/users/files/:email", controllers.GetUserFiles)
	router.GET("/files/:id", controllers.GetFileByID)
	router.DELETE("/files/:id", controllers.DeleteFile)

	// Lamport clock — lets other nodes (or a monitor) read this node's logical time
	router.GET("/clock", controllers.GetClock)

	router.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.String(400, "Get form err: %s", err.Error())
			return
		}
		os.MkdirAll("./uploads", os.ModePerm)
		if err := c.SaveUploadedFile(file, "./uploads/"+file.Filename); err != nil {
			c.String(400, "Upload err: %s", err.Error())
			return
		}
		c.JSON(200, gin.H{"message": "success"})
	})

	router.GET("/files", func(c *gin.Context) {
		var fileNames []string
		entries, _ := os.ReadDir("./uploads")
		for _, e := range entries {
			if !e.IsDir() {
				fileNames = append(fileNames, e.Name())
			}
		}
		if fileNames == nil {
			fileNames = make([]string, 0)
		}
		c.JSON(200, fileNames)
	})

	router.GET("/download", func(c *gin.Context) {
		fileName := c.Query("file")
		c.File("./uploads/" + fileName)
	})

	router.GET("/status", em.HandleStatus)
	router.POST("/shutdown", em.HandleShutdown)

	router.GET("/election/status", em.HandleStatus)
	router.POST("/election/resign", em.HandleResign)


	router.Run(":" + config.Load().Port)
}



