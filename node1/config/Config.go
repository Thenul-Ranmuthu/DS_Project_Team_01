package config

import (
	"os"
	"strings"
)

type Config struct {
	NodeID    string
	Port      string
	ZKServers []string
	Peers     []string
}

func Load() Config {
	peersStr := os.Getenv("PEERS")
	var peers []string

	if peersStr != "" {
		for _, p := range strings.Split(peersStr, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				peers = append(peers, trimmed)
			}
		}
	}

	return Config{
		NodeID:    os.Getenv("NODE_ID"),                       // e.g. "node-5050"
		Port:      os.Getenv("PORT"),                          // e.g. "5050"
		ZKServers: []string{os.Getenv("ZK_SERVER") + ":8080"}, // same for all nodes
		Peers:     peers,
	}
}
