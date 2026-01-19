package db

import (
	"context"
	"database/sql"
	_ "embed"
	"log"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

func Init(ctx context.Context) *sql.DB {
	database, err := sql.Open("sqlite", "./nyct.db")
	if err != nil {
		log.Fatal(err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=OFF",
		"PRAGMA synchronous=OFF",
	}

	for _, pragma := range pragmas {
		if _, err := database.ExecContext(ctx, pragma); err != nil {
			log.Fatal(err)
		}
	}

	// create tables
	if _, err := database.ExecContext(ctx, schema); err != nil {
		log.Fatal(err)
	}

	return database
}
