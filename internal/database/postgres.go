package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// DB wraps sql.DB with application-specific helpers
type DB struct {
	*sql.DB
	log *zap.Logger
}

// New creates a new database connection with migrations
func New(postgresURL string, log *zap.Logger) (*DB, error) {
	// Run migrations first
	if err := runMigrations(postgresURL); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create connection pool
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("database connection established")

	return &DB{
		DB:  db,
		log: log,
	}, nil
}

// runMigrations runs database migrations by reading SQL files from the migrations directory
func runMigrations(postgresURL string) error {
	// Open database connection
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer db.Close()

	// Find migrations directory (try multiple paths for flexibility)
	migrationsDir := ""
	possiblePaths := []string{"migrations", "/app/migrations", "./migrations"}
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			migrationsDir = path
			break
		}
	}
	if migrationsDir == "" {
		return fmt.Errorf("migrations directory not found")
	}

	// Read migration files
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort SQL files
	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	// Track applied migrations using a simple approach
	// Check if migrations table exists
	_, err = db.Exec("SELECT 1 FROM schema_migrations LIMIT 1")
	migrationsTableExists := err == nil

	// Create migrations table if it doesn't exist
	if !migrationsTableExists {
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version VARCHAR(255) PRIMARY KEY
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create migrations table: %w", err)
		}
	}

	// Apply each migration
	for _, sqlFile := range sqlFiles {
		// Extract version from filename (e.g., "001_init.sql" -> "001")
		version := strings.TrimSuffix(sqlFile, ".sql")
		if idx := strings.Index(version, "_"); idx != -1 {
			version = version[:idx]
		}

		// Check if already applied
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}
		if exists {
			continue // Already applied
		}

		// Read and execute migration
		migrationPath := filepath.Join(migrationsDir, sqlFile)
		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", sqlFile, err)
		}

		// Execute migration in transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		_, err = tx.Exec(string(migrationSQL))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", sqlFile, err)
		}

		// Record migration
		_, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", sqlFile, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", sqlFile, err)
		}
	}

	return nil
}

// WithTenant sets the tenant context for row-level security
func (db *DB) WithTenant(ctx context.Context, tenantID string) context.Context {
	// This would set app.current_tenant for RLS
	// Implementation depends on how you want to handle tenant context
	return ctx
}

// Close gracefully closes the database connection
func (db *DB) Close() error {
	db.log.Info("closing database connection")
	return db.DB.Close()
}
