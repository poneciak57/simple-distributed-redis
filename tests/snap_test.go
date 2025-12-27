package tests

import (
	"main/src"
	"os"
	"testing"
)

func RunSnapshotterTest_SnapshotAndLoad(t *testing.T,
	createSnapshotter func() (src.Snapshoter[string], func()),
	createWal func() (src.Wal[string], func())) {

	wal, cleanupWal := createWal()
	defer cleanupWal()
	defer wal.Close()

	// Add some entries to WAL
	wal.Append(src.WalEntry[string]{OpType: src.SET, Key: "key1", Value: "val1"}, true)
	wal.Append(src.WalEntry[string]{OpType: src.SET, Key: "key2", Value: "val2"}, true)
	wal.Append(src.WalEntry[string]{OpType: src.DELETE, Key: "key1"}, true)

	snapper, cleanupSnap := createSnapshotter()
	defer cleanupSnap()

	// Take Snapshot
	if err := snapper.Snapshot(wal); err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// Load Snapshot
	store, err := snapper.LoadSnapshot()
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	// Verify content
	// key1 should be deleted
	exists, _ := store.Exists("key1")
	if exists {
		t.Error("key1 should not exist")
	}

	// key2 should exist
	val, _ := store.Get("key2")
	if val != "val2" {
		t.Errorf("Expected key2=val2, got %s", val)
	}
}

func RunSnapshotterTest_IncrementalSnapshot(t *testing.T,
	createSnapshotter func() (src.Snapshoter[string], func()),
	createWal func() (src.Wal[string], func())) {

	snapper, cleanupSnap := createSnapshotter()
	defer cleanupSnap()

	// 1. WAL 1
	wal1, cleanupWal1 := createWal()
	wal1.Append(src.WalEntry[string]{OpType: src.SET, Key: "base", Value: "data"}, true)

	if err := snapper.Snapshot(wal1); err != nil {
		t.Fatal(err)
	}
	wal1.Close()
	cleanupWal1()

	// 2. WAL 2
	wal2, cleanupWal2 := createWal()
	wal2.Append(src.WalEntry[string]{OpType: src.SET, Key: "new", Value: "stuff"}, true)
	wal2.Append(src.WalEntry[string]{OpType: src.DELETE, Key: "base"}, true)

	if err := snapper.Snapshot(wal2); err != nil {
		t.Fatal(err)
	}
	wal2.Close()
	cleanupWal2()

	// Verify final state
	store, err := snapper.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}

	if exists, _ := store.Exists("base"); exists {
		t.Error("base key should be deleted")
	}
	if val, _ := store.Get("new"); val != "stuff" {
		t.Errorf("Expected new=stuff, got %s", val)
	}
}

func TestSimpleSnapshotter(t *testing.T) {
	createWal := func() (src.Wal[string], func()) {
		f, err := os.CreateTemp("", "wal_test_*.log")
		if err != nil {
			t.Fatal(err)
		}
		name := f.Name()
		f.Close()

		wal, err := src.NewSimpleWal(name)
		if err != nil {
			t.Fatal(err)
		}
		return wal, func() {
			os.Remove(name)
		}
	}

	createSnapshotter := func() (src.Snapshoter[string], func()) {
		f, err := os.CreateTemp("", "snap_test_*.bin")
		if err != nil {
			t.Fatal(err)
		}
		name := f.Name()
		f.Close()
		os.Remove(name) // Ensure it doesn't exist initially

		snapper := src.NewSimpleSnapshotter[string](name)
		return snapper, func() {
			os.Remove(name)
		}
	}

	t.Run("SnapshotAndLoad", func(t *testing.T) {
		RunSnapshotterTest_SnapshotAndLoad(t, createSnapshotter, createWal)
	})

	t.Run("IncrementalSnapshot", func(t *testing.T) {
		RunSnapshotterTest_IncrementalSnapshot(t, createSnapshotter, createWal)
	})
}
