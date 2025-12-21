package tests

import (
	"fmt"
	"main/src"
	"sync"
	"testing"
)

// RunStorageTests defines the contract tests for the Storage interface.
// Any implementation of Storage should pass these tests.
func RunStorageTests(t *testing.T, s src.Storage[string]) {
	t.Helper()

	t.Run("Set and Get", func(t *testing.T) {
		key := "testKey"
		value := "testValue"

		if err := s.Set(key, value); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		retrievedValue, err := s.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if retrievedValue != value {
			t.Errorf("Expected %s, got %s", value, retrievedValue)
		}
	})

	t.Run("Exists", func(t *testing.T) {
		key := "existsKey"
		value := "existsValue"

		// Should not exist yet
		exists, err := s.Exists(key)
		if err != nil {
			t.Fatalf("Exists check failed: %v", err)
		}
		if exists {
			t.Error("Key should not exist yet")
		}

		// Set and check again
		if err := s.Set(key, value); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		exists, err = s.Exists(key)
		if err != nil {
			t.Fatalf("Exists check failed: %v", err)
		}
		if !exists {
			t.Error("Key should exist")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		key := "deleteKey"
		value := "deleteValue"

		if err := s.Set(key, value); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		if err := s.Delete(key); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		exists, err := s.Exists(key)
		if err != nil {
			t.Fatalf("Exists check failed: %v", err)
		}
		if exists {
			t.Error("Key should have been deleted")
		}
	})

	t.Run("BulkSet", func(t *testing.T) {
		data := []struct {
			Key   string
			Value string
		}{
			{"bulkKey1", "bulkValue1"},
			{"bulkKey2", "bulkValue2"},
			{"bulkKey3", "bulkValue3"},
		}

		if err := s.BulkSet(data); err != nil {
			t.Fatalf("BulkSet failed: %v", err)
		}

		for _, item := range data {
			retrievedValue, err := s.Get(item.Key)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if retrievedValue != item.Value {
				t.Errorf("Expected %s, got %s", item.Value, retrievedValue)
			}
		}
	})

	t.Run("Get non-existing key", func(t *testing.T) {
		key := "nonExistingKey"

		value, err := s.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if value != "" {
			t.Errorf("Expected empty value for non-existing key, got %s", value)
		}
	})

	t.Run("Check multithreading safety", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100
		numOperations := 100
		wg.Add(numGoroutines * 2)
		// Writer goroutines
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("key-%d-%d", id, j)
					value := fmt.Sprintf("value-%d-%d", id, j)
					if err := s.Set(key, value); err != nil {
						t.Errorf("Set failed: %v", err)
					}
				}
			}(i)
		}

		// Reader goroutines
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("key-%d-%d", id, j)
					_, err := s.Get(key)
					if err != nil {
						t.Errorf("Get failed: %v", err)
					}
				}
			}(i)
		}

		wg.Wait()
	})
}

func TestInMemoryStorage(t *testing.T) {
	storage := src.MakeInMemoryStorage[string]()
	RunStorageTests(t, storage)
}
