package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"main/src/protocol"
	"os"
)

type Snapshoter[T any] interface {
	LoadSnapshot() (Storage[T], error)
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
	_, err := os.Stat(s.snapshotPath)
	if os.IsNotExist(err) {
		return MakeInMemoryStorage[T](), nil
	}
	if err != nil {
		return nil, err
	}

	fd, err := os.Open(s.snapshotPath)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	store := MakeInMemoryStorage[T]()

	for {
		var size int64
		err := binary.Read(fd, binary.LittleEndian, &size)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		data := make([]byte, size)
		_, err = io.ReadFull(fd, data)
		if err != nil {
			return nil, err
		}

		var entry SnapshotEntry[T]
		if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&entry); err != nil {
			return nil, err
		}
		store.Set(entry.Key, entry.Value)
	}

	return store, nil
}

func (s *SimpleSnapshotter[T]) Snapshot(wal Wal[T]) error {
	cur, err := s.LoadSnapshot()
	if err != nil {
		return err
	}

	if err := modify_store(wal, cur); err != nil {
		return err
	}

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

		payload := buf.Bytes()
		size := int64(len(payload))

		if err := binary.Write(fd, binary.LittleEndian, size); err != nil {
			return err
		}

		// Write to file
		_, err := fd.Write(payload)
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

func modify_store[T any](wal Wal[T], store Storage[T]) error {
	entries, err := wal.Replay()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		switch entry.OpType {
		case protocol.GET:
			// No-op for snapshot
		case protocol.SET:
			if err := store.Set(entry.Key, entry.Value); err != nil {
				return err
			}
		case protocol.DELETE:
			if err := store.Delete(entry.Key); err != nil {
				return err
			}
		case protocol.PING:
			// No-op for snapshot
		default:
			return fmt.Errorf("unknown operation type in WAL: %v", entry.OpType)
		}
	}

	return nil
}
