package storage

// IMPORTANT! storage is not intended to be thread safe
type Storage[T any] interface {
	Get(key string) (T, error)
	Set(key string, value T) error
	Delete(key string) error
	Exists(key string) (bool, error)

	// Iterator returns a function that iterates over all key-value pairs in storage.
	// It uses experimental generators feature.
	Iterator() func(func(string, T) bool)
}

// Simple in memory implementation of storage
type InMemoryStorage[T any] struct {
	data map[string]T
}

func MakeInMemoryStorage[T any]() *InMemoryStorage[T] {
	return &InMemoryStorage[T]{
		data: make(map[string]T),
	}
}

func (s *InMemoryStorage[T]) Get(key string) (T, error) {
	value, exists := s.data[key]
	if !exists {
		var zero T
		return zero, nil
	}
	return value, nil
}

func (s *InMemoryStorage[T]) Set(key string, value T) error {
	s.data[key] = value
	return nil
}

func (s *InMemoryStorage[T]) Exists(key string) (bool, error) {
	_, exists := s.data[key]
	return exists, nil
}

func (s *InMemoryStorage[T]) Delete(key string) error {
	delete(s.data, key)
	return nil
}

func (s *InMemoryStorage[T]) Iterator() func(func(string, T) bool) {
	return func(yield func(string, T) bool) {
		for k := range s.data {
			if !yield(k, s.data[k]) {
				break
			}
		}
	}
}
