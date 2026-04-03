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
	// All nodes maintain their own independent DB, so all schemas are migrated universally.
	initializers.DB.AutoMigrate(&models.UploadedFile{})
	initializers.DB.AutoMigrate(&models.User{})
	initializers.DB.AutoMigrate(&models.ElectionEvent{})
	initializers.DB.AutoMigrate(&models.IdempotencyRecord{})
	initializers.DB.AutoMigrate(&models.WriteAheadLog{})
	initializers.DB.AutoMigrate(&models.ReplicationState{})
}
