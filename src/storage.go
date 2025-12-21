package src

import "sync"

type Storage[T any] interface {
	Get(key string) (T, error)
	Set(key string, value T) error
	Delete(key string) error
	Exists(key string) (bool, error)
	BulkSet(data []struct {
		Key   string
		Value T
	}) error
}

type InMemoryStorage[T any] struct {
	mu   sync.RWMutex
	data map[string]T
}

func MakeInMemoryStorage[T any]() *InMemoryStorage[T] {
	return &InMemoryStorage[T]{
		mu:   sync.RWMutex{},
		data: make(map[string]T),
	}
}

func (s *InMemoryStorage[T]) Get(key string) (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.data[key]
	if !exists {
		var zero T
		return zero, nil
	}
	return value, nil
}

func (s *InMemoryStorage[T]) Set(key string, value T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *InMemoryStorage[T]) BulkSet(data []struct {
	Key   string
	Value T
}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range data {
		s.data[item.Key] = item.Value
	}
	return nil
}

func (s *InMemoryStorage[T]) Exists(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.data[key]
	return exists, nil
}

func (s *InMemoryStorage[T]) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}
