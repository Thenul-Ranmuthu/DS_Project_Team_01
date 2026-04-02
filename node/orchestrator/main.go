package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/gin-gonic/gin"
	"runtime"
)

type NodeProcess struct {
	ID   string
	Port string
	Cmd  *exec.Cmd
}

var (
	nodes     = make(map[string]*NodeProcess)
	mu        sync.Mutex
	nodeCount = 7
)

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

func startNode(id string, port string) error {
	mu.Lock()
	defer mu.Unlock()

	if p, ok := nodes[id]; ok && p.Cmd != nil && p.Cmd.Process != nil {
		return fmt.Errorf("node %s already running", id)
	}

	binaryName := "./server"
	if runtime.GOOS == "windows" {
		binaryName = "./server.exe"
	}
	cmd := exec.Command(binaryName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%s", port),
		fmt.Sprintf("NODE_ID=%s", id),
	)

	err := cmd.Start()
	if err != nil {
		return err
	}

	nodes[id] = &NodeProcess{
		ID:   id,
		Port: port,
		Cmd:  cmd,
	}

	log.Printf("[Orchestrator] Started %s on port %s", id, port)
	return nil
}

func stopNode(id string) error {
	mu.Lock()
	defer mu.Unlock()

	p, ok := nodes[id]
	if !ok || p.Cmd == nil || p.Cmd.Process == nil {
		return fmt.Errorf("node %s not running", id)
	}

	err := p.Cmd.Process.Kill()
	if err != nil {
		return err
	}

	p.Cmd.Wait() // wait for cleanup
	p.Cmd = nil
	log.Printf("[Orchestrator] Stopped %s", id)
	return nil
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(CORSMiddleware())

	// Initialize nodes
	for i := 1; i <= nodeCount; i++ {
		id := fmt.Sprintf("node_%d", i)
		port := fmt.Sprintf("%d", 8000+i-1)
		err := startNode(id, port)
		if err != nil {
			log.Printf("[ERROR] Initial start failed for %s: %v", id, err)
		}
	}

	router.POST("/shutdown/:id", func(c *gin.Context) {
		id := c.Param("id")
		err := stopNode(id)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "node stopped"})
	})

	router.POST("/recover/:id", func(c *gin.Context) {
		id := c.Param("id")
		// Extract port from id (e.g., node_1 -> 8000)
		var port int
		fmt.Sscanf(id, "node_%d", &port)
		portStr := fmt.Sprintf("%d", 8000+port-1)

		err := startNode(id, portStr)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "node recovered"})
	})

	router.GET("/status", func(c *gin.Context) {
		mu.Lock()
		defer mu.Unlock()
		status := make(map[string]string)
		for id, p := range nodes {
			if p.Cmd != nil && p.Cmd.Process != nil {
				status[id] = "RUNNING"
			} else {
				status[id] = "STOPPED"
			}
		}
		c.JSON(200, status)
	})

	log.Println("[Orchestrator] Control Service listening on :9999")
	router.Run(":9999")
}
