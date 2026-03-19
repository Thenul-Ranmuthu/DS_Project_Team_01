package main

import (
	controllers "github.com/DS_node/Controllers"
	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/migrate"
	"github.com/gin-gonic/gin"
)

func init() {
	initializers.LoadEnvVaribles()
	migrate.MigrateDB()
}

func main() {
	router := gin.Default()

	router.POST("/createUser", controllers.CreateUser)

	router.GET("/ping", controllers.PingEndPoint)

	router.POST("/upload/:email", controllers.UploadMultipleFiles)
	router.GET("/users/files/:email", controllers.GetUserFiles)
	router.GET("/files/:id", controllers.GetFileByID)
	router.DELETE("/files/:id", controllers.DeleteFile)

	// Lamport clock — lets other nodes (or a monitor) read this node's logical time
	router.GET("/clock", controllers.GetClock)

	router.Run()
}
