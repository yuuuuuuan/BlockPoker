package storage

import (
	"database/sql"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitPostgres(dsn string) error {
	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	return DB.Ping()
}
