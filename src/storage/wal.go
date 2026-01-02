package storage

import (
	"fmt"
	"io"
	"main/src/protocol"
	"os"
	"path/filepath"
	"time"
)

// Term represents a logical time period in the Raft consensus algorithm.
// It is a monotonically increasing integer that allows nodes to detect
// stale information (e.g., from an old leader).
type Term uint64

// WalEntry represents a single log entry in the Write-Ahead Log.
type WalEntry[T any] struct {
	Index     uint64 // Monotonically increasing index
	Timestamp int64
	Term      Term
	OpType    protocol.OpType
	Key       string
	Value     T
}

// Wal (Write-Ahead Log) interface defines the methods for durability.
// This interface is not thread-safe, caller should rotate the log it should allow concurrent access.
type Wal[T any] interface {
	// Append adds a new entry to the log.
	Append(entry WalEntry[T], sync bool) error

	// Replay reads the log from the beginning and returns all entries.
	// Used for restoring state on startup.
	Replay() ([]WalEntry[T], error)

	// Rotates the log, so it clears resources and returns old log handle.
	// Caller is responsible for closing the returned Wal.
	Rotate() (Wal[T], error)

	// Size returns the current size of the WAL file in bytes.
	Size() int64

	// Close closes the underlying file or resource.
	Close() error
}

// SimpleWal is a basic implementation of the Wal interface.
type SimpleWal[T any] struct {
	filePath string
	fd       *os.File
	size     int64
}

// NewSimpleWal creates a new instance of SimpleWal.
func NewSimpleWal[T any](filePath string) (*SimpleWal[T], error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	fd, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	stat, err := fd.Stat()
	if err != nil {
		fd.Close()
		return nil, err
	}

	return &SimpleWal[T]{
		filePath: filePath,
		fd:       fd,
		size:     stat.Size(),
	}, nil
}

func NewSimpleWalFromFile[T any](fd *os.File) (*SimpleWal[T], error) {
	stat, err := fd.Stat()
	if err != nil {
		return nil, err
	}

	return &SimpleWal[T]{
		filePath: fd.Name(),
		fd:       fd,
		size:     stat.Size(),
	}, nil
}

// Close closes the underlying file.
func (w *SimpleWal[T]) Close() error {
	if w.fd != nil {
		return w.fd.Close()
	}
	return nil
}

// Size returns the current size of the WAL file in bytes.
func (w *SimpleWal[T]) Size() int64 {
	return w.size
}

func (w *SimpleWal[T]) Append(entry WalEntry[T], sync bool) error {
	// Serialize entry to RESP2 Array
	// [Index, Timestamp, Term, OpType, Key, Value]
	arr := []protocol.Resp2Value{
		protocol.Resp2Integer(entry.Index),
		protocol.Resp2Integer(entry.Timestamp),
		protocol.Resp2Integer(entry.Term),
		protocol.Resp2Integer(entry.OpType),
		protocol.Resp2BulkString(entry.Key),
		entry.Value,
	}

	parser := protocol.NewResp2Parser(nil, 0)
	payload, err := parser.Render(arr)
	if err != nil {
		return err
	}

	// Write to file
	n, err := w.fd.Write(payload)
	if err != nil {
		return err
	}

	w.size += int64(n)

	if sync {
		return w.fd.Sync()
	}
	return nil
}

// Replay reads the log from the beginning and returns all entries.
func (w *SimpleWal[T]) Replay() ([]WalEntry[T], error) {
	// Seek to the beginning of the file
	if _, err := w.fd.Seek(0, 0); err != nil {
		return nil, err
	}

	var entries []WalEntry[T]
	parser := protocol.NewResp2Parser(w.fd, 0)

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
			return nil, fmt.Errorf("invalid WAL entry format: expected array")
		}

		if len(arr) != 6 {
			return nil, fmt.Errorf("invalid WAL entry format: expected 6 elements, got %d", len(arr))
		}

		index, ok := arr[0].(protocol.Resp2Integer)
		if !ok {
			return nil, fmt.Errorf("invalid WAL entry format: expected integer for Index")
		}

		timestamp, ok := arr[1].(protocol.Resp2Integer)
		if !ok {
			return nil, fmt.Errorf("invalid WAL entry format: expected integer for Timestamp")
		}

		term, ok := arr[2].(protocol.Resp2Integer)
		if !ok {
			return nil, fmt.Errorf("invalid WAL entry format: expected integer for Term")
		}

		opType, ok := arr[3].(protocol.Resp2Integer)
		if !ok {
			return nil, fmt.Errorf("invalid WAL entry format: expected integer for OpType")
		}

		key, ok := arr[4].(protocol.Resp2BulkString)
		if !ok {
			return nil, fmt.Errorf("invalid WAL entry format: expected bulk string for Key")
		}

		value := arr[5]

		// Cast value to T
		// Since T is likely Resp2Value (interface{}), this should work.
		// If T is a concrete type, we might have issues if value is not that type.
		// But we assume T is Resp2Value.
		var tValue T
		if value != nil {
			var ok bool
			tValue, ok = value.(T)
			if !ok {
				return nil, fmt.Errorf("invalid WAL entry format: expected value of type %T", *new(T))
			}
		}

		entries = append(entries, WalEntry[T]{
			Index:     uint64(index),
			Timestamp: int64(timestamp),
			Term:      Term(term),
			OpType:    protocol.OpType(opType),
			Key:       string(key),
			Value:     tValue,
		})
	}
	return entries, nil
}

// Rotate closes the current file, renames it to a backup/old path, and opens a fresh file at the original path.
// Returns the file handle to the rotated file (caller must close it).
func (w *SimpleWal[T]) Rotate() (Wal[T], error) {
	// 1. Sync current
	if err := w.fd.Sync(); err != nil {
		return nil, err
	}

	// 2. Keep reference to old fd
	oldFd := w.fd

	// 3. Rename file on disk
	// We rename the file that w.fd points to.
	// Note: On Linux/Unix, w.fd will still point to the renamed file (inode).
	timestamp := time.Now().UnixNano()
	rotatedPath := fmt.Sprintf("%s.%d", w.filePath, timestamp)

	if err := os.Rename(w.filePath, rotatedPath); err != nil {
		return nil, err
	}

	// 4. Open new file at original path
	newFd, err := os.OpenFile(w.filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		// If we fail to open new file, we are in a weird state.
		// We might want to try to rename back, but for now return error.
		return nil, err
	}

	w.fd = newFd
	w.size = 0

	oldFd.Seek(0, 0) // Reset old file descriptor to beginning for reading
	// Return the old file handle (which now points to rotatedPath)
	return NewSimpleWalFromFile[T](oldFd)
}
