package config

import "os"

type Config struct {
	NodeID    string
	Port      string
	ZKServers []string
	Peers     []string
}

func Load() Config {
	zk := os.Getenv("ZK_SERVERS")
	if zk == "" {
		zk = "172.30.112.1:2181" // Default IP
	} else if zk == "local" {
		zk = "localhost:2181"
	}

	return Config{
		NodeID:    os.Getenv("NODE_ID"),
		Port:      os.Getenv("PORT"),
		ZKServers: []string{zk},
	}
}

