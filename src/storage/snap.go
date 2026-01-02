package storage

import (
	"fmt"
	"io"
	"main/src/protocol"
	"os"
	"path/filepath"
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
	parser := protocol.NewResp2Parser(fd, 0)

	for {
		val, err := parser.Parse()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		arr, ok := val.([]protocol.Resp2Value)
		if !ok {
			return nil, fmt.Errorf("invalid snapshot entry format: expected array")
		}

		if len(arr) != 2 {
			return nil, fmt.Errorf("invalid snapshot entry format: expected 2 elements, got %d", len(arr))
		}

		key, ok := arr[0].(protocol.Resp2BulkString)
		if !ok {
			return nil, fmt.Errorf("invalid snapshot entry format: expected bulk string for Key")
		}

		value := arr[1]
		var tValue T
		if value != nil {
			var ok bool
			tValue, ok = value.(T)
			if !ok {
				return nil, fmt.Errorf("invalid snapshot entry format: expected value of type %T", *new(T))
			}
		}

		store.Set(string(key), tValue)
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
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0755); err != nil {
		return err
	}

	// Create temp file
	fd, err := os.OpenFile(snapshotPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()

	parser := protocol.NewResp2Parser(nil, 0)

	// Write all key-value pairs to snapshot file
	var writeErr error
	store.Iterator()(func(k string, v T) bool {
		arr := []protocol.Resp2Value{
			protocol.Resp2BulkString(k),
			v,
		}
		payload, err := parser.Render(arr)
		if err != nil {
			writeErr = err
			return false
		}

		// Write to file
		_, err = fd.Write(payload)
		if err != nil {
			writeErr = err
			return false
		}
		return true
	})

	if writeErr != nil {
		return writeErr
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
