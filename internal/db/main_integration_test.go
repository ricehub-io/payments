//go:build integration

package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testDB *Database

func TestMain(m *testing.M) {
	ctx := context.Background()

	migrationsDir := filepath.Join("..", "..", "migrations")

	pgC, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("payments_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.WithOrderedInitScripts(
			filepath.Join(migrationsDir, "001_initial.sql"),
			filepath.Join(migrationsDir, "002_add_webhook_events_table.sql"),
			filepath.Join(migrationsDir, "003_add_status_column_to_subscriptions.sql"),
		),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		panic("postgres.Run: " + err.Error())
	}

	connStr, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("ConnectionString: " + err.Error())
	}

	testDB, err = NewDatabase(connStr)
	if err != nil {
		panic("NewDatabase: " + err.Error())
	}

	code := m.Run()

	testDB.Close()
	_ = pgC.Terminate(ctx)

	os.Exit(code)
}
