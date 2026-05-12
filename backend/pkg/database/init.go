package database

import (
	"database/sql"
	_ "embed"
)

//go:embed schema.sql
var schemaSQL string

func InitSchema(db *DB) error {
	_, err := db.Exec(schemaSQL)
	return err
}
