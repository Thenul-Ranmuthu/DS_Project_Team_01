package config

import (
	"os"
	"strings"
)

type Config struct {
	NodeID          string
	Port            string
	ZKServers       []string
	Peers           []string
	MinioEndpoint   string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioBucketName string
	MinioUseSSL     bool
}

func Load() Config {
	zk := strings.TrimSpace(os.Getenv("ZK_SERVERS"))
	if zk == "" {
		zk = "172.30.112.1:2181" // Default IP
	} else if zk == "local" {
		zk = "localhost:2181"
	}

	useSSL := strings.ToLower(os.Getenv("MINIO_USE_SSL")) == "true"

	return Config{
		NodeID:          strings.TrimSpace(os.Getenv("NODE_ID")),
		Port:            strings.TrimSpace(os.Getenv("PORT")),
		ZKServers:       []string{zk},
		MinioEndpoint:   os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey:  os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:  os.Getenv("MINIO_SECRET_KEY"),
		MinioBucketName: os.Getenv("MINIO_BUCKET_NAME"),
		MinioUseSSL:     useSSL,
	}
}


