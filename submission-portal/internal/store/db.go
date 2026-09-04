// Package store provides the persistence layer supporting Turso/libSQL and local SQLite.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql" // Turso / libSQL driver
	_ "modernc.org/sqlite"                               // Pure-Go SQLite driver (no CGO)
)

// DB wraps the sql.DB handle and stores connection metadata.
type DB struct {
	*sql.DB
	path     string
	isRemote bool
}

// ErrNotFound is returned when a queried row does not exist.
var ErrNotFound = errors.New("not found")

// IsRemote reports whether the connected database is a remote libSQL/Turso cluster.
func (db *DB) IsRemote() bool { return db.isRemote }

// Path returns the database file path or remote connection URL.
func (db *DB) Path() string { return db.path }

// Open opens a database connection with default authentication.
func Open(target string) (*DB, error) {
	return OpenWithAuth(target, "")
}

// OpenWithAuth opens a connection to either a remote Turso/libSQL instance or a local SQLite database.
func OpenWithAuth(target, authToken string) (*DB, error) {
	if target == "" {
		target = "ctf.db"
	}

	isRemote := strings.HasPrefix(target, "libsql://") ||
		strings.HasPrefix(target, "https://") ||
		strings.HasPrefix(target, "http://") ||
		strings.HasPrefix(target, "ws://") ||
		strings.HasPrefix(target, "wss://")

	if authToken == "" {
		authToken = os.Getenv("TURSO_AUTH_TOKEN")
		if authToken == "" {
			authToken = os.Getenv("CTF_DB_AUTH_TOKEN")
		}
		if authToken == "" {
			authToken = os.Getenv("LIBSQL_AUTH_TOKEN")
		}
	}

	var sqldb *sql.DB
	var err error

	if isRemote {
		dsn := target
		if authToken != "" && !strings.Contains(dsn, "authToken=") {
			sep := "?"
			if strings.Contains(dsn, "?") {
				sep = "&"
			}
			dsn = fmt.Sprintf("%s%sauthToken=%s", dsn, sep, authToken)
		}
		sqldb, err = sql.Open("libsql", dsn)
		if err != nil {
			return nil, fmt.Errorf("open libsql (%s): %w", target, err)
		}
		sqldb.SetMaxOpenConns(25)
		sqldb.SetMaxIdleConns(5)
		sqldb.SetConnMaxLifetime(10 * time.Minute)
	} else {
		path := target
		if strings.HasPrefix(path, "file:") {
			path = strings.TrimPrefix(path, "file:")
			if idx := strings.Index(path, "?"); idx != -1 {
				path = path[:idx]
			}
		}
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create db dir: %w", err)
			}
		}
		dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)", path)
		sqldb, err = sql.Open("sqlite", dsn)
		if err != nil {
			return nil, fmt.Errorf("open sqlite (%s): %w", path, err)
		}
		sqldb.SetMaxOpenConns(1)
	}

	if err := sqldb.Ping(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("ping database (%s): %w", target, err)
	}

	return &DB{DB: sqldb, path: target, isRemote: isRemote}, nil
}

// MustOpen is Open but exits the process on failure.
func MustOpen(target string) *DB {
	return MustOpenWithAuth(target, "")
}

// MustOpenWithAuth is OpenWithAuth but exits the process on failure.
func MustOpenWithAuth(target, authToken string) *DB {
	db, err := OpenWithAuth(target, authToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: open database %s: %v\n", target, err)
		os.Exit(1)
	}
	return db
}

// splitMigrationStatements splits SQL content into executable statements,
// filtering comments and PRAGMAs.
func splitMigrationStatements(content string) []string {
	var cleanLines []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(trimmed), "PRAGMA ") {
			continue
		}
		cleanLines = append(cleanLines, line)
	}

	raw := strings.Join(cleanLines, "\n")
	var stmts []string
	for _, part := range strings.Split(raw, ";") {
		stmt := strings.TrimSpace(part)
		if stmt != "" {
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}

// Migrate applies all pending .sql migrations from dir, in filename order,
// tracking applied versions in schema_migrations.
func (db *DB) Migrate(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %q: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	for _, name := range files {
		var done int
		if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, name).Scan(&done); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if done > 0 {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		stmts := splitMigrationStatements(string(body))
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}
		for idx, stmt := range stmts {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("apply migration %s (statement #%d): %w\nSQL: %s", name, idx+1, err, stmt)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, name); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}

// Backup writes a consistent snapshot of the database to dest via VACUUM INTO.
func (db *DB) Backup(dest string) error {
	if db.isRemote {
		return fmt.Errorf("backup via VACUUM INTO is not supported for remote Turso/libSQL databases; use Turso Cloud automated backups or replicas")
	}
	safe := strings.ReplaceAll(dest, "'", "''")
	if _, err := db.Exec(fmt.Sprintf("VACUUM INTO '%s'", safe)); err != nil {
		return fmt.Errorf("vacuum into %s: %w", dest, err)
	}
	return nil
}
