package postgres

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type StorageClient struct {
	DB *sql.DB
}

// NewClient PostgreSQL Instance
func NewClient(dsn string) (*StorageClient, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection pool: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database unreachable: %w", err)
	}

	log.Println("[Storage-Postgres] Connected to relational database successfully.")
	return &StorageClient{DB: db}, nil
}

// Close the connection pool securely on shutdown
func (c *StorageClient) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}
