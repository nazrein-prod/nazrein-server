package store

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func ConnectClickhouse() (driver.Conn, error) {

	context := context.Background()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"localhost:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "password",
		},
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: "trackyt-clickhouse-api-server", Version: "1.0"},
			},
		},
		Debugf: func(format string, v ...interface{}) {
			fmt.Printf(format, v)
		},
	})

	if err != nil {
		return nil, fmt.Errorf("db: open %w", err)
	}

	err = conn.Ping(context)
	if err != nil {
		return nil, fmt.Errorf("db: open %w", err)
	}

	fmt.Println("Connected to Clickhouse...")
	return conn, nil
}

func MigrateClickhouse() error {

	migrationURL := "file://./migrations/analytics"
	dbURL := "clickhouse://default:password@localhost:9000/default?x-multi-statement=true"

	m, err := migrate.New(migrationURL, dbURL)
	if err != nil {
		return fmt.Errorf("migration init error: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %v", err)
	}

	return nil
}
