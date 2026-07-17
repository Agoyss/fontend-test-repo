package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// Env returns the environment variable or a fallback default.
func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadDotEnv reads a .env file (KEY=VALUE lines) into the process
// environment. Shell-provided variables take precedence and missing keys
// are ignored, so this is safe to call even if no .env exists.
func LoadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // no .env file: rely on real env / defaults
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		val = strings.Trim(val, `"'`) // strip optional quotes
		if key != "" && os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

// dsn builds the MySQL connection string from environment overrides.
func dsn() string {
	user := Env("DB_USER", "root")
	pass := os.Getenv("DB_PASSWORD")
	host := Env("DB_HOST", "127.0.0.1")
	port := Env("DB_PORT", "3306")
	name := Env("DB_NAME", "article")
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, name)
}

// Connect opens the database and verifies the connection.
func Connect() (*sql.DB, error) {
	LoadDotEnv(".env")
	db, err := sql.Open("mysql", dsn())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cannot connect to database: %w", err)
	}
	return db, nil
}

// applyMigrations runs every *.up.sql file in dir once. Already-applied
// migrations (tracked in schema_migrations) are skipped so the app can boot
// repeatedly without "table already exists" errors.
func applyMigrations(db *sql.DB, dir string) error {
	if _, err := db.Exec(
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			name VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	); err != nil {
		return fmt.Errorf("cannot init migrations table: %w", err)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var ups []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".up.sql") {
			ups = append(ups, f.Name())
		}
	}
	sort.Strings(ups)

	for _, name := range ups {
		var count int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM schema_migrations WHERE name = ?", name,
		).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			fmt.Println("skip migration (already applied):", name)
			continue
		}
		sqlText, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(sqlText)); err != nil {
			return fmt.Errorf("migration %s failed: %w", name, err)
		}
		if _, err := db.Exec(
			"INSERT INTO schema_migrations (name) VALUES (?)", name,
		); err != nil {
			return fmt.Errorf("migration tracking %s failed: %w", name, err)
		}
		fmt.Println("applied migration:", name)
	}
	return nil
}

// Migrate applies all pending migrations from the migrations directory.
func Migrate(db *sql.DB, dir string) error {
	return applyMigrations(db, dir)
}

// Reset drops all tables and the migration tracker so the next Migrate
// re-applies everything from scratch.
func Reset(db *sql.DB) error {
	fmt.Println("RESET_DB: dropping tables + migration tracking")
	for _, t := range []string{"articles", "trash", "schema_migrations"} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + t); err != nil {
			return err
		}
	}
	return nil
}
