package database

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

func InitDB(connStr string) error {
	var err error
	DB, err = sql.Open("pgx", connStr)
	if err != nil {
		return err
	}
	return DB.Ping()
}

func PingDB() error {
	return DB.Ping()
}
