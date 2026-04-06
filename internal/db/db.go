package db

import (
	"fmt"

	"finance-backend-challenge/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func Connect(cfg *config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("sqlite3", cfg.DBURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
