package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/boltdb/bolt"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

func main() {
	nodeID := flag.String("node-id", "node1", "Raft node ID")
	raftDir := flag.String("raft-dir", "./data", "Directory for raft data")
	raftAddr := flag.String("raft-addr", ":9000", "Raft communication address")
	httpAddr := flag.String("http-addr", ":8000", "HTTP server address")
	bootstrap := flag.Bool("bootstrap", false, "Bootstrap as leader")
	joinAddr := flag.String("join", "", "Address of leader to join")
	flag.Parse()

	os.MkdirAll(*raftDir, 0700)

	dbPath := filepath.Join(*raftDir, "files.db")
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatal("failed to open bolt DB: ", err)
	}
	defer db.Close()
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("files"))
		return err
	})
	store := NewStore(db)

	logStorePath := filepath.Join(*raftDir, "raft-log.db")
	stableStorePath := filepath.Join(*raftDir, "raft-stable.db")

	logStore, err := raftboltdb.NewBoltStore(logStorePath)
	if err != nil {
		log.Fatal(err)
	}
	
	stableStore, err := raftboltdb.NewBoltStore(stableStorePath)
	if err != nil {
		log.Fatal(err)
	}
	
	snapshotStore, err := raft.NewFileSnapshotStore(*raftDir, 1, os.Stderr)
	if err != nil {
		log.Fatal(err)
	}

	raftConf := raft.DefaultConfig()
	raftConf.LocalID = raft.ServerID(*nodeID)

	// Use 127.0.0.1 for local testing to avoid 'not advertisable' errors
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1"+*raftAddr)
	if err != nil {
		log.Fatal(err)
	}
	transport, err := raft.NewTCPTransport("127.0.0.1"+*raftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		log.Fatal(err)
	}

	fsm := &FileFSM{store: store}
	r, err := raft.NewRaft(raftConf, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		log.Fatal(err)
	}

	if *bootstrap {
		conf := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raftConf.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		r.BootstrapCluster(conf)
		log.Println("Bootstrapping cluster...")
	} else if *joinAddr != "" {
		go func() {
			time.Sleep(3 * time.Second)
			// Ensure we point to the leader on localhost
			joinURL := fmt.Sprintf("http://localhost%s/join?id=%s&addr=127.0.0.1%s", *joinAddr, *nodeID, *raftAddr)
			resp, err := http.Get(joinURL)
			if err != nil {
				log.Printf("Failed to join cluster: %v", err)
			} else {
				resp.Body.Close()
				log.Printf("Requested join to %s", *joinAddr)
			}
		}()
	}

	StartHttpServer(*httpAddr, r, store)
	log.Printf("Node %s running HTTP on %s, Raft on %s", *nodeID, *httpAddr, *raftAddr)

	select {}
}
