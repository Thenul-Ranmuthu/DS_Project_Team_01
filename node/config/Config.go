package config

import "os"

type Config struct {
	NodeID    string
	Port      string
	ZKServers []string
	Peers     []string
}

func Load() Config {
	return Config{
		NodeID:    os.Getenv("NODE_ID"),             // e.g. "node-5050"
		Port:      os.Getenv("PORT"),                // e.g. "5050"
		ZKServers: []string{os.Getenv("ZK_SERVER")}, // same for all nodes
	}
}
