package database

import (
	"database/sql"
	"os"
)

func InitSchema(db *DB) error {
	schema, err := os.ReadFile("pkg/database/schema.sql")
	if err != nil {
		return err
	}

	_, err = db.Exec(string(schema))
	return err
}
