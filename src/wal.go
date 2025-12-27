package src

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
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
	OpType    OpType
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
type SimpleWal struct {
	filePath string
	fd       *os.File
	size     int64
}

// NewSimpleWal creates a new instance of SimpleWal.
func NewSimpleWal(filePath string) (*SimpleWal, error) {
	fd, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	stat, err := fd.Stat()
	if err != nil {
		fd.Close()
		return nil, err
	}

	return &SimpleWal{
		filePath: filePath,
		fd:       fd,
		size:     stat.Size(),
	}, nil
}

func NewSimpleWalFromFile(fd *os.File) (*SimpleWal, error) {
	stat, err := fd.Stat()
	if err != nil {
		return nil, err
	}

	return &SimpleWal{
		filePath: fd.Name(),
		fd:       fd,
		size:     stat.Size(),
	}, nil
}

// Close closes the underlying file.
func (w *SimpleWal) Close() error {
	if w.fd != nil {
		return w.fd.Close()
	}
	return nil
}

// Size returns the current size of the WAL file in bytes.
func (w *SimpleWal) Size() int64 {
	return w.size
}

func (w *SimpleWal) Append(entry WalEntry[string], sync bool) error {
	var buf bytes.Buffer
	// Use gob for easy serialization
	if err := gob.NewEncoder(&buf).Encode(entry); err != nil {
		return err
	}

	// Write to file
	n, err := w.fd.Write(buf.Bytes())
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
func (w *SimpleWal) Replay() ([]WalEntry[string], error) {
	// Seek to the beginning of the file
	if _, err := w.fd.Seek(0, 0); err != nil {
		return nil, err
	}

	var entries []WalEntry[string]
	decoder := gob.NewDecoder(w.fd)

	for {
		var entry WalEntry[string]
		err := decoder.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			// If we encounter an error (e.g. partial write at the end),
			// we might want to stop and return what we have, or return error.
			// For now, return error.
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// Rotate closes the current file, renames it to a backup/old path, and opens a fresh file at the original path.
// Returns the file handle to the rotated file (caller must close it).
func (w *SimpleWal) Rotate() (Wal[string], error) {
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
	return NewSimpleWalFromFile(oldFd)
}
