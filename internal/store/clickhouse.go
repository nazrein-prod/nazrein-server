package store

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func ConnectClickhouse() (driver.Conn, error) {
	ctx := context.Background()
	var conn driver.Conn
	var err error

	url := os.Getenv("CLICKHOUSE_URL")
	dbName := os.Getenv("CLICKHOUSE_DATABASE")
	username := os.Getenv("CLICKHOUSE_USERNAME")
	password := os.Getenv("CLICKHOUSE_PASSWORD")

	for i := 1; i <= 10; i++ {
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{url},
			Auth: clickhouse.Auth{
				Database: dbName,
				Username: username,
				Password: password,
			},
			ClientInfo: clickhouse.ClientInfo{
				Products: []struct {
					Name    string
					Version string
				}{
					{Name: "nazrein-clickhouse-api-server", Version: "1.0"},
				},
			},
			Debugf: func(format string, v ...interface{}) {
				fmt.Printf(format, v)
			},
		})

		if err == nil {
			err = conn.Ping(ctx)
			if err == nil {
				fmt.Println("Connected to ClickHouse!")
				return conn, nil
			}
		}

		fmt.Printf("Attempt %d: ClickHouse not ready: %v\n", i, err)
		time.Sleep(3 * time.Second)
	}

	return nil, fmt.Errorf("could not connect to ClickHouse after multiple attempts: %w", err)
}

func MigrateClickhouse() error {
	migrationURL := "file://./migrations/analytics"

	username := os.Getenv("CLICKHOUSE_USERNAME")
	password := os.Getenv("CLICKHOUSE_PASSWORD")
	url := os.Getenv("CLICKHOUSE_URL")
	dbName := os.Getenv("CLICKHOUSE_DATABASE")

	dbURL := fmt.Sprintf("clickhouse://%s:%s@%s/%s?x-multi-statement=true",
		username, password, url, dbName)

	m, err := migrate.New(migrationURL, dbURL)
	if err != nil {
		return fmt.Errorf("migration init error: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %v", err)
	}

	return nil
}
