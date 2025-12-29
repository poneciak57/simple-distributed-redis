package tests

import (
	"bytes"
	"fmt"
	"io"
	"main/src/config"
	"main/src/service"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// MockConn implements net.Conn for testing
type MockConn struct {
	readBuf   *bytes.Buffer
	writeBuf  *bytes.Buffer
	chunkSize int // If > 0, Read will return at most chunkSize bytes
	closed    bool
}

func NewMockConn(data []byte) *MockConn {
	return &MockConn{
		readBuf:  bytes.NewBuffer(data),
		writeBuf: new(bytes.Buffer),
	}
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	if m.closed {
		return 0, io.EOF
	}
	if m.chunkSize > 0 {
		if len(b) > m.chunkSize {
			b = b[:m.chunkSize]
		}
	}
	return m.readBuf.Read(b)
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.writeBuf.Write(b)
}

func (m *MockConn) Close() error {
	m.closed = true
	return nil
}

func (m *MockConn) LocalAddr() net.Addr                { return nil }
func (m *MockConn) RemoteAddr() net.Addr               { return nil }
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

func setupTestRedis(t *testing.T) (*service.RedisService, string) {
	tmpDir, err := os.MkdirTemp("", "redis_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Snapshot.Path = filepath.Join(tmpDir, "snapshot.db")
	cfg.WAL.Path = filepath.Join(tmpDir, "wal.log")

	storageSvc := service.NewStorageService(cfg)
	redisSvc := service.NewRedisServices(storageSvc, cfg)

	return redisSvc, tmpDir
}

func TestRedisService_SimpleCommands(t *testing.T) {
	svc, tmpDir := setupTestRedis(t)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "PING",
			input:    "*1\r\n$4\r\nPING\r\n",
			expected: "+PONG\r\n",
		},
		{
			name:     "SET key val",
			input:    "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$3\r\nval\r\n",
			expected: "+OK\r\n",
		},
		{
			name:     "GET key",
			input:    "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n",
			expected: "$3\r\nval\r\n",
		},
		{
			name:     "DELETE key",
			input:    "*2\r\n$6\r\nDELETE\r\n$3\r\nkey\r\n",
			expected: "+OK\r\n",
		},
		{
			name:     "GET deleted key",
			input:    "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n",
			expected: "$0\r\n\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := NewMockConn([]byte(tt.input))
			err := svc.OnMessage(conn)
			if err != nil {
				t.Fatalf("OnMessage failed: %v", err)
			}
			if got := conn.writeBuf.String(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRedisService_ChunkyReads(t *testing.T) {
	svc, tmpDir := setupTestRedis(t)
	defer os.RemoveAll(tmpDir)

	input := "*3\r\n$3\r\nSET\r\n$6\r\nchunky\r\n$5\r\nvalue\r\n"
	expected := "+OK\r\n"

	conn := NewMockConn([]byte(input))
	conn.chunkSize = 2 // Read 2 bytes at a time

	err := svc.OnMessage(conn)
	if err != nil {
		t.Fatalf("OnMessage failed with chunky reads: %v", err)
	}

	if got := conn.writeBuf.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestRedisService_ConcurrentConnections(t *testing.T) {
	svc, tmpDir := setupTestRedis(t)
	defer os.RemoveAll(tmpDir)

	concurrency := 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			val := fmt.Sprintf("val-%d", id)

			// SET
			setInput := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(val), val)
			conn := NewMockConn([]byte(setInput))
			if err := svc.OnMessage(conn); err != nil {
				t.Errorf("Concurrent SET failed: %v", err)
				return
			}
			if got := conn.writeBuf.String(); got != "+OK\r\n" {
				t.Errorf("Concurrent SET expected +OK, got %q", got)
				return
			}

			// GET
			getInput := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
			conn = NewMockConn([]byte(getInput))
			if err := svc.OnMessage(conn); err != nil {
				t.Errorf("Concurrent GET failed: %v", err)
				return
			}
			expected := fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
			if got := conn.writeBuf.String(); got != expected {
				t.Errorf("Concurrent GET expected %q, got %q", expected, got)
			}
		}(i)
	}

	wg.Wait()
}
