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
	// 1. Get the raw string from .env (e.g. "http://localhost:5051")
	peersStr := os.Getenv("PEERS")
	var peers []string
	
	if peersStr != "" {
		// 2. Split the string by commas in case you have multiple peers
		peers = strings.Split(peersStr, ",")
	}

	return Config{
		NodeID:    os.Getenv("NODE_ID"),
		Port:      os.Getenv("PORT"),
		ZKServers: []string{"127.0.0.1:2181"}, // Updated to your ZK address
		Peers:     peers,                     // Now this is populated!
	}
}