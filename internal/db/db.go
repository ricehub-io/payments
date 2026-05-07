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
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, connUrl)
	if err != nil {
		return nil, fmt.Errorf("pgxpool new: %w", err)
	}

	return &Database{pool}, nil
}

func (d *Database) Close() {
	d.pool.Close()
}

// -- HELPERS --
// type DBExecutor interface {
// 	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
// 	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
// 	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
// }

// func rowToStruct[T any](ctx context.Context, exec DBExecutor, query string, args ...any) (res T, err error) {
// 	rows, err := exec.Query(ctx, query, args...)
// 	if err != nil {
// 		return res, err
// 	}

// 	return pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
// }

// func rowsToStruct[T any](ctx context.Context, exec DBExecutor, query string, args ...any) (res []T, err error) {
// 	rows, err := exec.Query(ctx, query, args...)
// 	if err != nil {
// 		return res, err
// 	}

// 	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
// }
