package db

import (
	"context"
	"database/sql"
	"fmt"
)

type DB struct {
	db *sql.DB
}

func New(db *sql.DB) *DB {
	return &DB{db: db}
}

func prKey(project, repo string, id int) string {
	return fmt.Sprintf("%s/%s/%d", project, repo, id)
}

func (db *DB) HasPullRequest(ctx context.Context, project, repo string, id int) (bool, error) {
	var exists bool
	key := prKey(project, repo, id)

	row := db.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT key FROM pulls WHERE key = ?)", key)
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("determining if row exists: %w", err)
	}

	return exists, nil
}

func (db *DB) LastActivity(ctx context.Context, project, repo string, id int) (lastActivity int, err error) {
	key := prKey(project, repo, id)
	row := db.db.QueryRowContext(ctx, "SELECT last_activity FROM pulls WHERE key = ?", key)
	if err := row.Scan(&lastActivity); err != nil {
		return 0, fmt.Errorf("determining last activity: %w", err)
	}

	return lastActivity, nil
}

func (db *DB) UpsertPullRequest(ctx context.Context, project, repo string, id, lastActivty int) error {
	key := prKey(project, repo, id)
	_, err := db.db.ExecContext(ctx, `
INSERT INTO pulls (key, last_activity)
VALUES (?, ?) ON CONFLICT(key) DO
  UPDATE SET last_activity = excluded.last_activity
`,
		key, lastActivty,
	)

	if err != nil {
		return fmt.Errorf("upserting pull request: %w", err)
	}

	return nil
}
