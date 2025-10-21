package main

import (
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var (
		dsn             string
		migrationsPath  string
		migrationsTable string
	)

	flag.StringVar(&dsn, "dsn", "", "PostgreSQL DSN, e.g. postgres://user:pass@localhost:5432/db?sslmode=disable")
	flag.StringVar(&migrationsPath, "migrations-path", "", "path to migrations directory")
	flag.StringVar(&migrationsTable, "migrations-table", "schema_migrations", "name of migrations table")
	flag.Parse()

	if dsn == "" {
		log.Fatal("dsn is required")
	}
	if migrationsPath == "" {
		log.Fatal("migrations-path is required")
	}

	sep := "?"
	if hasQuery(dsn) {
		sep = "&"
	}
	dsnWithTable := fmt.Sprintf("%s%sx-migrations-table=%s", dsn, sep, migrationsTable)

	m, err := migrate.New(
		"file://"+migrationsPath,
		dsnWithTable,
	)
	if err != nil {
		log.Fatalf("init migrate: %v", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("no migrations to apply")
			return
		}
		log.Fatalf("apply migrations: %v", err)
	}

	log.Println("migrations applied")
}

func hasQuery(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '?' {
			return true
		}
	}
	return false
}
