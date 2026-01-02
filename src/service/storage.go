package service

import (
	"main/src/config"
	"main/src/protocol"
	"main/src/storage"
	"sync"
	"time"
)

// Responsible for handling storage related services
// makes snapshots, manages storage and wal
// TODO it might be temporary or will be changed after consensus implementation
// Ideal implementation will not use mutex but rely more on channels
type StorageService struct {
	wal          storage.Wal[protocol.Resp2Value]
	snapshotter  storage.Snapshoter[protocol.Resp2Value]
	storage      storage.Storage[protocol.Resp2Value]
	cfg          *config.Config
	logger       *config.Logger
	mu           sync.RWMutex
	lastSnapTime int64
}

func NewStorageService(config *config.Config, logger *config.Logger) *StorageService {
	snapshotter := storage.NewSimpleSnapshotter[protocol.Resp2Value](config.Snapshot.Path)
	wal, err := storage.NewSimpleWal[protocol.Resp2Value](config.WAL.Path)
	if err != nil {
		logger.Error("Failed to create WAL: %v", err)
		panic(err)
	}
	storageInstance, err := snapshotter.LoadSnapshot()
	if err != nil {
		logger.Error("Failed to load snapshot: %v", err)
		panic(err)
	}
	entries, err := wal.Replay()
	if err != nil {
		logger.Error("Failed to replay WAL: %v", err)
		panic(err)
	}

	// TODO might require some change after consensus implementation
	for _, entry := range entries {
		storageInstance.Set(entry.Key, entry.Value)
	}

	return &StorageService{
		wal:          wal,
		snapshotter:  snapshotter,
		storage:      storageInstance,
		cfg:          config,
		logger:       logger,
		lastSnapTime: time.Now().Unix(),
		mu:           sync.RWMutex{},
	}
}

func (s *StorageService) Snapshot() error {
	rotatedWal, err := s.wal.Rotate()
	if err != nil {
		return err
	}
	err = s.snapshotter.Snapshot(rotatedWal)
	if err != nil {
		return err
	}
	return rotatedWal.Close()
}

func (s *StorageService) SnapshotIfNeeded() error {
	if s.wal.Size() >= s.cfg.Snapshot.Threshold {
		return s.Snapshot()
	} else if time.Now().Unix()-s.lastSnapTime >= int64(s.cfg.Snapshot.Interval) {
		s.lastSnapTime = time.Now().Unix()
		return s.Snapshot()
	}
	return nil
}

func (s *StorageService) Set(key string, value protocol.Resp2Value) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.wal.Append(storage.WalEntry[protocol.Resp2Value]{
		Key:   key,
		Value: value,
	}, true)
	if err != nil {
		return err
	}
	return s.storage.Set(key, value)
}

func (s *StorageService) Get(key string) (protocol.Resp2Value, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.storage.Get(key)
}

func (s *StorageService) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.wal.Append(storage.WalEntry[protocol.Resp2Value]{
		Key:   key,
		Value: nil,
	}, true)
	if err != nil {
		return err
	}
	return s.storage.Delete(key)
}

func (s *StorageService) Exists(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.storage.Exists(key)
}
