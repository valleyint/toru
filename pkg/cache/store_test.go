package cache

import (
	"os"
	"testing"
)

func setupTestDB(t *testing.T) (*Store, string) {
	// Create a temp file for sqlite DB
	f, err := os.CreateTemp("", "toru-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	f.Close()

	store, err := NewStore(f.Name())
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	return store, f.Name()
}

func teardownTestDB(store *Store, path string) {
	store.Close()
	os.Remove(path)
}

func TestTranslationCache(t *testing.T) {
	s, path := setupTestDB(t)
	defer teardownTestDB(s, path)

	// Test Miss
	trans, master, err := s.GetTranslation("neovim", "nix_unstable")
	if err != nil {
		t.Fatalf("expected nil error on miss, got %v", err)
	}
	if trans != "" || master != "" {
		t.Errorf("expected empty strings on miss")
	}

	// Test Save
	err = s.SaveTranslation("neovim", "nix_unstable", "neovim-unwrapped", "neovim")
	if err != nil {
		t.Fatalf("failed to save translation: %v", err)
	}

	// Test Hit
	trans, master, err = s.GetTranslation("neovim", "nix_unstable")
	if err != nil {
		t.Fatalf("expected nil error on hit, got %v", err)
	}
	if trans != "neovim-unwrapped" || master != "neovim" {
		t.Errorf("got %s, %s; want neovim-unwrapped, neovim", trans, master)
	}
}

func TestDependencyLifecycle(t *testing.T) {
	s, path := setupTestDB(t)
	defer teardownTestDB(s, path)

	// Install a parent explicitly
	s.MarkInstalled("htop", "nix", true, false)
	
	// Install a child as a dependency
	s.MarkInstalled("ncurses", "nix", false, false)
	
	// Link them
	err := s.AddDependency("htop", "ncurses", "nix")
	if err != nil {
		t.Fatalf("failed to add dependency: %v", err)
	}

	// Check orphan status of child
	isOrphan, err := s.IsOrphan("ncurses", "nix")
	if err != nil {
		t.Fatalf("failed to check orphan: %v", err)
	}
	if isOrphan {
		t.Errorf("expected ncurses to NOT be an orphan because it has a parent")
	}

	// Remove the parent
	err = s.RemovePackage("htop", "nix")
	if err != nil {
		t.Fatalf("failed to remove parent: %v", err)
	}

	// Check orphan status of child again
	isOrphan, err = s.IsOrphan("ncurses", "nix")
	if err != nil {
		t.Fatalf("failed to check orphan: %v", err)
	}
	if !isOrphan {
		t.Errorf("expected ncurses to be an orphan because its parent was removed and it's not explicit")
	}
}
