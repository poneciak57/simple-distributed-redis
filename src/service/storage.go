package service

import (
	"main/src/config"
	"main/src/storage"
	"time"
)

// Responsible for handling storage related services
// makes snapshots, manages storage and wal
// TODO it might be temporary or will be changed after consensus implementation
type StorageService struct {
	wal          storage.Wal[string]
	snapshotter  storage.Snapshoter[string]
	storage      storage.Storage[string]
	cfg          *config.Config
	lastSnapTime int64
}

func NewStorageService(config *config.Config) *StorageService {
	snapshotter := storage.NewSimpleSnapshotter[string](config.Snapshot.Path)
	wal, err := storage.NewSimpleWal(config.WAL.Path)
	if err != nil {
		panic(err)
	}
	storageInstance, err := snapshotter.LoadSnapshot()
	if err != nil {
		panic(err)
	}
	entries, err := wal.Replay()
	if err != nil {
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
		lastSnapTime: time.Now().Unix(),
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

func (s *StorageService) Set(key, value string) error {
	err := s.wal.Append(storage.WalEntry[string]{
		Key:   key,
		Value: value,
	}, true)
	if err != nil {
		return err
	}
	return s.storage.Set(key, value)
}

func (s *StorageService) Get(key string) (string, error) {
	return s.storage.Get(key)
}

func (s *StorageService) Delete(key string) error {
	err := s.wal.Append(storage.WalEntry[string]{
		Key:   key,
		Value: "",
	}, true)
	if err != nil {
		return err
	}
	return s.storage.Delete(key)
}

func (s *StorageService) Exists(key string) (bool, error) {
	return s.storage.Exists(key)
}
