package migrations

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed *.sql
var files embed.FS

const upMarker = "-- +imageflow Up"
const downMarker = "-- +imageflow Down"

type migration struct {
	version int64
	up      string
	down    string
}

func Run(ctx context.Context, databaseURL, direction string) error {
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer func() { _ = connection.Close(ctx) }()

	if _, err := connection.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	all, err := load()
	if err != nil {
		return err
	}
	if direction == "up" {
		return migrateUp(ctx, connection, all)
	}
	if direction == "down" {
		return migrateDown(ctx, connection, all)
	}
	return fmt.Errorf("unsupported migration direction %q", direction)
}

func migrateUp(ctx context.Context, connection *pgx.Conn, all []migration) error {
	for _, item := range all {
		var applied bool
		if err := connection.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, item.version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", item.version, err)
		}
		if applied {
			continue
		}
		if err := apply(ctx, connection, item.up, `INSERT INTO schema_migrations (version) VALUES ($1)`, item.version); err != nil {
			return fmt.Errorf("apply migration %d: %w", item.version, err)
		}
	}
	return nil
}

func migrateDown(ctx context.Context, connection *pgx.Conn, all []migration) error {
	var version int64
	err := connection.QueryRow(ctx, `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find latest migration: %w", err)
	}
	for _, item := range all {
		if item.version == version {
			if err := apply(ctx, connection, item.down, `DELETE FROM schema_migrations WHERE version = $1`, version); err != nil {
				return fmt.Errorf("revert migration %d: %w", version, err)
			}
			return nil
		}
	}
	return fmt.Errorf("migration file for version %d not found", version)
}

func apply(ctx context.Context, connection *pgx.Conn, sql, versionStatement string, version int64) error {
	transaction, err := connection.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err := transaction.Exec(ctx, sql, pgx.QueryExecModeSimpleProtocol); err != nil {
		return err
	}
	if _, err := transaction.Exec(ctx, versionStatement, version); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

func load() ([]migration, error) {
	entries, err := files.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	var result []migration
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		versionText, _, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}
		version, err := strconv.ParseInt(versionText, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid migration version in %q: %w", entry.Name(), err)
		}
		contents, err := files.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}
		parts := strings.SplitN(string(contents), downMarker, 2)
		if len(parts) != 2 || !strings.Contains(parts[0], upMarker) {
			return nil, fmt.Errorf("migration %q must contain up and down markers", entry.Name())
		}
		result = append(result, migration{
			version: version,
			up:      strings.TrimSpace(strings.Replace(parts[0], upMarker, "", 1)),
			down:    strings.TrimSpace(parts[1]),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].version < result[j].version })
	return result, nil
}
