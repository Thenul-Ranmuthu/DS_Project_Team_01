package initializers

import (
	"context"
	"log"

	"github.com/DS_node/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioClient *minio.Client

func InitStorage() {
	cfg := config.Load()

	// Initialize minio client object.
	var err error
	MinioClient, err = minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		log.Fatalln("Failed to initialize MinIO client:", err)
	}

	// Make a new bucket.
	bucketName := cfg.MinioBucketName
	location := "us-east-1"

	ctx := context.Background()
	err = MinioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		// Check to see if we already own this bucket (which is happens quite often)
		exists, errBucketExists := MinioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("Bucket %s already exists\n", bucketName)
		} else {
			log.Printf("Warning: Failed to create bucket %s or confirm its existence: %v\n", bucketName, err)
		}
	} else {
		log.Printf("Successfully created bucket %s\n", bucketName)
	}
}

func GetBucketName() string {
	return config.Load().MinioBucketName
}
