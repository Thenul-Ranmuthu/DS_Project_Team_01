package migrate

import (
	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/models"
)

func init() {
	initializers.LoadEnvVaribles()
	initializers.ConnectToDB()
}

func MigrateDB() {
	initializers.DB.AutoMigrate(&models.UploadedFile{})
	initializers.DB.AutoMigrate(&models.User{})
	initializers.DB.AutoMigrate(&models.ElectionEvent{})
	initializers.DB.AutoMigrate(&models.PendingReplication{})
}
