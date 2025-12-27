package tests

import (
	"main/src"
	"os"
	"testing"
)

func RunWalTest_AppendAndReplay(t *testing.T, setup func() (src.Wal[string], func() (src.Wal[string], error), func())) {
	wal, reopen, cleanup := setup()
	defer cleanup()

	entries := []src.WalEntry[string]{
		{Index: 1, OpType: src.SET, Key: "key1", Value: "val1"},
		{Index: 2, OpType: src.SET, Key: "key2", Value: "val2"},
		{Index: 3, OpType: src.DELETE, Key: "key1"},
	}

	for _, e := range entries {
		if err := wal.Append(e, true); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	if wal.Size() == 0 {
		t.Error("WAL size should not be 0")
	}

	wal.Close()

	// Reopen
	wal2, err := reopen()
	if err != nil {
		t.Fatal(err)
	}
	defer wal2.Close()

	replayed, err := wal2.Replay()
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if len(replayed) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(replayed))
	}

	for i, e := range replayed {
		if e.Index != entries[i].Index {
			t.Errorf("Entry %d: expected Index %d, got %d", i, entries[i].Index, e.Index)
		}
		if e.Key != entries[i].Key {
			t.Errorf("Entry %d: expected Key %s, got %s", i, entries[i].Key, e.Key)
		}
		if e.Value != entries[i].Value {
			t.Errorf("Entry %d: expected Value %s, got %s", i, entries[i].Value, e.Value)
		}
		if e.OpType != entries[i].OpType {
			t.Errorf("Entry %d: expected OpType %v, got %v", i, entries[i].OpType, e.OpType)
		}
	}
}

func RunWalTest_Rotate(t *testing.T, setup func() (src.Wal[string], func() (src.Wal[string], error), func())) {
	wal, _, cleanup := setup()
	defer cleanup()
	defer wal.Close()

	entry := src.WalEntry[string]{Index: 1, OpType: src.SET, Key: "k", Value: "v"}
	wal.Append(entry, true)

	oldWal, err := wal.Rotate()
	if err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}
	defer oldWal.Close()

	// Check old wal content
	oldEntries, err := oldWal.Replay()
	if err != nil {
		t.Fatalf("Old WAL replay failed: %v", err)
	}
	if len(oldEntries) != 1 {
		t.Errorf("Expected 1 entry in old WAL, got %d", len(oldEntries))
	}

	// Check new wal is empty
	if wal.Size() != 0 {
		t.Errorf("New WAL size should be 0, got %d", wal.Size())
	}

	// Append to new wal
	entry2 := src.WalEntry[string]{Index: 2, OpType: src.SET, Key: "k2", Value: "v2"}
	if err := wal.Append(entry2, true); err != nil {
		t.Fatalf("Append to new WAL failed: %v", err)
	}

	// Verify new wal content
	newEntries, err := wal.Replay()
	if err != nil {
		t.Fatalf("New WAL replay failed: %v", err)
	}
	if len(newEntries) != 1 {
		t.Errorf("Expected 1 entry in new WAL, got %d", len(newEntries))
	}
	if newEntries[0].Key != "k2" {
		t.Errorf("Expected key k2 in new WAL, got %s", newEntries[0].Key)
	}
}

func TestSimpleWal(t *testing.T) {
	setup := func() (src.Wal[string], func() (src.Wal[string], error), func()) {
		dir, err := os.MkdirTemp("", "wal_test")
		if err != nil {
			t.Fatal(err)
		}
		path := dir + "/wal.log"

		wal, err := src.NewSimpleWal(path)
		if err != nil {
			t.Fatal(err)
		}

		reopen := func() (src.Wal[string], error) {
			return src.NewSimpleWal(path)
		}

		cleanup := func() {
			wal.Close()
			os.RemoveAll(dir)
		}
		return wal, reopen, cleanup
	}

	t.Run("AppendAndReplay", func(t *testing.T) {
		RunWalTest_AppendAndReplay(t, setup)
	})

	t.Run("Rotate", func(t *testing.T) {
		RunWalTest_Rotate(t, setup)
	})
}
