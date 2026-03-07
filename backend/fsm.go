package main

import (
	"encoding/json"
	"io"

	"github.com/hashicorp/raft"
)

type Event struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value []byte `json:"value"`
}

type FileFSM struct {
	store *Store
}

func (f *FileFSM) Apply(log *raft.Log) interface{} {
	var e Event
	if err := json.Unmarshal(log.Data, &e); err != nil {
		return err
	}

	switch e.Op {
	case "SET":
		return f.store.Set(e.Key, e.Value)
	}
	return nil
}

func (f *FileFSM) Snapshot() (raft.FSMSnapshot, error) {
	return &fsmSnapshot{}, nil
}

func (f *FileFSM) Restore(rc io.ReadCloser) error {
	return nil
}

type fsmSnapshot struct{}

func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	return sink.Cancel()
}

func (f *fsmSnapshot) Release() {}
