package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	pool *pgxpool.Pool
}

func NewDatabase(connUrl string) (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, connUrl)
	if err != nil {
		return nil, fmt.Errorf("pgxpool new: %w", err)
	}

	// test query
	var two uint
	err = pool.QueryRow(ctx, "SELECT 2").Scan(&two)
	if err != nil {
		return nil, fmt.Errorf("test query scan: %w", err)
	}
	if two != 2 {
		return nil, fmt.Errorf("invalid test query result: got %d, expected 2", two)
	}

	return &Database{pool}, nil
}

func (d *Database) Close() {
	d.pool.Close()
}
