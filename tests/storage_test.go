package tests

import (
	"fmt"
	"main/src/storage"
	"testing"
)

// RunStorageTests defines the contract tests for the Storage interface.
// Any implementation of Storage should pass these tests.
func RunStorageTests(t *testing.T, s storage.Storage[string]) {
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

	t.Run("Manny Sets and gets", func(t *testing.T) {
		n := 1025
		for i := 0; i < n; i++ {
			key := fmt.Sprintf("key-%d", i)
			value := fmt.Sprintf("value-%d", i)
			if err := s.Set(key, value); err != nil {
				t.Fatalf("Set failed: %v", err)
			}
		}

		for i := 0; i < n; i++ {
			key := fmt.Sprintf("key-%d", i)
			value := fmt.Sprintf("value-%d", i)

			retrievedValue, err := s.Get(key)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if retrievedValue != value {
				t.Errorf("Expected %s, got %s", value, retrievedValue)
			}
		}
	})
}

func TestInMemoryStorage(t *testing.T) {
	storage := storage.MakeInMemoryStorage[string]()
	RunStorageTests(t, storage)
}
