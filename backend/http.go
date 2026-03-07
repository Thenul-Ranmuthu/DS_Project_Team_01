package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/raft"
)

type HttpServer struct {
	raft   *raft.Raft
	store  *Store
	nodeID string
}

func (s *HttpServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] Received UPLOAD request from %s", r.RemoteAddr)
	
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] Recovered in handleUpload: %v", r)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	if s.raft.State() != raft.Leader {
		leaderAddr, _ := s.raft.LeaderWithID()
		log.Printf("[HTTP] Rejecting upload: Not leader (current state: %s). Leader is: %s", s.raft.State(), leaderAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusLocked) // 423 Locked is often used in Distributed Systems for 'talk to leader'
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Not leader",
			"leader": string(leaderAddr),
		})
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		log.Printf("[HTTP] ParseMultipartForm error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("[HTTP] FormFile error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Printf("[HTTP] Uploading file: %s (%d bytes)", header.Filename, header.Size)

	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("[HTTP] ReadAll error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	event := Event{
		Op:    "SET",
		Key:   header.Filename,
		Value: data,
	}
	b, err := json.Marshal(event)
	if err != nil {
		log.Printf("[HTTP] Marshal error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[HTTP] Applying file to Raft cluster...")
	// Increased timeout to 10 seconds for larger files and replication
	f := s.raft.Apply(b, 10*time.Second)
	if f.Error() != nil {
		log.Printf("[RAFT] Apply error: %v", f.Error())
		http.Error(w, f.Error().Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[SUCCESS] File %s replicated successfully", header.Filename)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Uploaded %s\n", header.Filename)
}

func (s *HttpServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	if filename == "" {
		http.Error(w, "missing file arg", http.StatusBadRequest)
		return
	}

	val, err := s.store.Get(filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.WriteHeader(http.StatusOK)
	w.Write(val)
}

func (s *HttpServer) handleList(w http.ResponseWriter, r *http.Request) {
	keys, err := s.store.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

func (s *HttpServer) handleJoin(w http.ResponseWriter, r *http.Request) {
	if s.raft.State() != raft.Leader {
		http.Error(w, "Not leader", http.StatusBadRequest)
		return
	}

	nodeID := r.URL.Query().Get("id")
	raftAddr := r.URL.Query().Get("addr")
	if nodeID == "" || raftAddr == "" {
		http.Error(w, "missing id or addr", http.StatusBadRequest)
		return
	}

	f := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(raftAddr), 0, 0)
	if f.Error() != nil {
		http.Error(w, f.Error().Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *HttpServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	leaderAddr, leaderID := s.raft.LeaderWithID()
	stats := s.raft.Stats()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_id":        s.nodeID,
		"state":          s.raft.State().String(),
		"leader_addr":    string(leaderAddr),
		"leader_id":      string(leaderID),
		"commit_index":   stats["commit_index"],
		"applied_index":  stats["applied_index"],
		"last_log_index": stats["last_log_index"],
		"num_peers":      stats["num_peers"],
		"term":           stats["term"],
	})
}

func (s *HttpServer) handleShutdown(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] Remote shutdown requested...")
	w.WriteHeader(http.StatusOK)
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
}

func applyCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func StartHttpServer(addr string, r *raft.Raft, store *Store, nodeID string) {
	server := &HttpServer{raft: r, store: store, nodeID: nodeID}
	mux := http.NewServeMux()
	mux.HandleFunc("/upload", applyCORS(server.handleUpload))
	mux.HandleFunc("/download", applyCORS(server.handleDownload))
	mux.HandleFunc("/files", applyCORS(server.handleList))
	mux.HandleFunc("/join", applyCORS(server.handleJoin))
	mux.HandleFunc("/status", applyCORS(server.handleStatus))
	mux.HandleFunc("/shutdown", applyCORS(server.handleShutdown))
	
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}
	}()
}
