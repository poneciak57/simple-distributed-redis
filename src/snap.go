package src

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
)

type Snapshoter[T any] interface {
	LoadSnapshot(log *os.File) (Storage[T], error)
	Snapshot(wal Wal[T]) error
}

type SnapshotEntry[T any] struct {
	Key   string
	Value T
}

type SimpleSnapshotter[T any] struct {
	snapshotPath string
}

func NewSimpleSnapshotter[T any](snapshotPath string) *SimpleSnapshotter[T] {
	return &SimpleSnapshotter[T]{
		snapshotPath: snapshotPath,
	}
}

func (s *SimpleSnapshotter[T]) LoadSnapshot() (Storage[T], error) {
	return nil, nil
}

func (s *SimpleSnapshotter[T]) Snapshot(wal Wal[T]) error {
	cur, err := s.LoadSnapshot()
	if err != nil {
		return err
	}

	modify_store(wal, &cur)

	tmp_path := s.snapshotPath + ".tmp"
	err = snapshot(tmp_path, cur)
	if err != nil {
		return err
	}

	return os.Rename(tmp_path, s.snapshotPath)
}

func snapshot[T any](snapshotPath string, store Storage[T]) error {
	// Create temp file
	fd, err := os.OpenFile(snapshotPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()

	// Write all key-value pairs to snapshot file
	for k, v := range store.Iterator() {
		entry := SnapshotEntry[T]{
			Key:   k,
			Value: v,
		}
		var buf bytes.Buffer
		// Use gob for easy serialization
		if err := gob.NewEncoder(&buf).Encode(entry); err != nil {
			return err
		}

		// Write to file
		_, err := fd.Write(buf.Bytes())
		if err != nil {
			return err
		}
	}
	// Sync to ensure data is flushed
	if err := fd.Sync(); err != nil {
		return err
	}
	return nil
}

func modify_store[T any](wal Wal[T], store *Storage[T]) error {
	entries, err := wal.Replay()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		switch entry.OpType {
		case GET:
			// No-op for snapshot
		case SET:
			if err := (*store).Set(entry.Key, entry.Value); err != nil {
				return err
			}
		case DELETE:
			if err := (*store).Delete(entry.Key); err != nil {
				return err
			}
		case PING:
			// No-op for snapshot
		default:
			return fmt.Errorf("unknown operation type in WAL: %v", entry.OpType)
		}
	}

	return nil
}
