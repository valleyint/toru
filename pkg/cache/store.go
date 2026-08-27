package cache

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store provides the local SQLite database cache for Toru.
type Store struct {
	db *sql.DB
}

// NewStore initializes a connection to the SQLite database and ensures schemas exist.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys for cascading deletes
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	store := &Store{db: db}
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS translations (
			arch_name TEXT NOT NULL,
			backend TEXT NOT NULL,
			translated_name TEXT NOT NULL,
			master_name TEXT NOT NULL,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY (arch_name, backend)
		);`,
		`CREATE TABLE IF NOT EXISTS installed_packages (
			arch_name TEXT NOT NULL,
			backend TEXT NOT NULL,
			is_explicit BOOLEAN NOT NULL DEFAULT 0,
			is_parity_sync BOOLEAN NOT NULL DEFAULT 0,
			installed_at INTEGER NOT NULL,
			PRIMARY KEY (arch_name, backend)
		);`,
		`CREATE TABLE IF NOT EXISTS package_dependencies (
			parent_arch_name TEXT NOT NULL,
			child_arch_name TEXT NOT NULL,
			backend TEXT NOT NULL,
			PRIMARY KEY (parent_arch_name, child_arch_name, backend),
			FOREIGN KEY(parent_arch_name, backend) REFERENCES installed_packages(arch_name, backend) ON DELETE CASCADE,
			FOREIGN KEY(child_arch_name, backend) REFERENCES installed_packages(arch_name, backend) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS path_binaries (
			binary_name TEXT NOT NULL,
			arch_name TEXT NOT NULL,
			PRIMARY KEY (binary_name, arch_name)
		);`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// --- Translation Caching ---

// GetTranslation retrieves a cached mapping for an arch package.
func (s *Store) GetTranslation(archName, backend string) (string, string, error) {
	var translatedName, masterName string
	query := `SELECT translated_name, master_name FROM translations WHERE arch_name = ? AND backend = ?`
	err := s.db.QueryRow(query, archName, backend).Scan(&translatedName, &masterName)
	if err == sql.ErrNoRows {
		return "", "", nil // Cache miss
	}
	if err != nil {
		return "", "", err
	}
	return translatedName, masterName, nil
}

// SaveTranslation stores a translation to avoid future Repology API calls.
func (s *Store) SaveTranslation(archName, backend, translatedName, masterName string) error {
	query := `INSERT INTO translations (arch_name, backend, translated_name, master_name, updated_at) 
			  VALUES (?, ?, ?, ?, ?) 
			  ON CONFLICT(arch_name, backend) DO UPDATE SET 
			  translated_name=excluded.translated_name, 
			  master_name=excluded.master_name, 
			  updated_at=excluded.updated_at`
	_, err := s.db.Exec(query, archName, backend, translatedName, masterName, time.Now().Unix())
	return err
}

// --- Installation & Lifecycle Tracking ---

// MarkInstalled adds or updates a package in the tracker.
func (s *Store) MarkInstalled(archName, backend string, isExplicit, isParitySync bool) error {
	query := `INSERT INTO installed_packages (arch_name, backend, is_explicit, is_parity_sync, installed_at) 
			  VALUES (?, ?, ?, ?, ?) 
			  ON CONFLICT(arch_name, backend) DO UPDATE SET 
			  is_explicit=excluded.is_explicit, 
			  is_parity_sync=excluded.is_parity_sync`
	_, err := s.db.Exec(query, archName, backend, isExplicit, isParitySync, time.Now().Unix())
	return err
}

// RemovePackage deletes a package and cascades to remove its dependency links.
func (s *Store) RemovePackage(archName, backend string) error {
	query := `DELETE FROM installed_packages WHERE arch_name = ? AND backend = ?`
	_, err := s.db.Exec(query, archName, backend)
	return err
}

// AddDependency maps a child package to a parent package. Both must exist in installed_packages.
func (s *Store) AddDependency(parentArch, childArch, backend string) error {
	query := `INSERT INTO package_dependencies (parent_arch_name, child_arch_name, backend) 
			  VALUES (?, ?, ?) ON CONFLICT DO NOTHING`
	_, err := s.db.Exec(query, parentArch, childArch, backend)
	return err
}

// IsOrphan determines if a package can be safely uninstalled.
// It is an orphan if it has 0 parents AND was not explicitly requested or parity synced.
func (s *Store) IsOrphan(archName, backend string) (bool, error) {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM package_dependencies WHERE child_arch_name = ? AND backend = ?) as parent_count,
			is_explicit,
			is_parity_sync
		FROM installed_packages
		WHERE arch_name = ? AND backend = ?`
	
	var parentCount int
	var isExplicit, isParitySync bool

	err := s.db.QueryRow(query, archName, backend, archName, backend).Scan(&parentCount, &isExplicit, &isParitySync)
	if err == sql.ErrNoRows {
		return false, nil // Not tracked, safe to assume false
	}
	if err != nil {
		return false, err
	}

	return parentCount == 0 && !isExplicit && !isParitySync, nil
}

// --- Parity Binary Tracking ---

// AddBinary tracks an executable found in PATH mapped to its provider.
func (s *Store) AddBinary(binaryName, archName string) error {
	query := `INSERT INTO path_binaries (binary_name, arch_name) VALUES (?, ?) ON CONFLICT DO NOTHING`
	_, err := s.db.Exec(query, binaryName, archName)
	return err
}

// RemoveBinariesForPackage drops binary tracking when a package is removed.
func (s *Store) RemoveBinariesForPackage(archName string) error {
	query := `DELETE FROM path_binaries WHERE arch_name = ?`
	_, err := s.db.Exec(query, archName)
	return err
}
