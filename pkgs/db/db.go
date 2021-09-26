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

func (db *DB) HasPullRequest(ctx context.Context, project, repo string, id int) (bool, error) {
	var exists bool
	row := db.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT id FROM pulls WHERE project = ? AND repo = ? AND pr_id = ?)", project, repo, id)
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("determining if row exists: %w", err)
	}

	return exists, nil
}

func (db *DB) UpsertPullRequest(ctx context.Context, project, repo string, id, lastActivty int) error {
	_, err := db.db.ExecContext(ctx, "INSERT INTO pulls (project, repo, pr_id, last_activity) values (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET last_activity = excluded.last_activity",
		project, repo, id, lastActivty,
	)

	if err != nil {
		return fmt.Errorf("upserting pull request: %w", err)
	}

	return nil
}
