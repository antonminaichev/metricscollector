package database

import (
	"database/sql"
	"fmt"

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

func InitMetricsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS metrics (
			id VARCHAR NOT NULL,
			type VARCHAR NOT NULL,
			delta BIGINT,
			value DOUBLE PRECISION,
			PRIMARY KEY (id, type)
		)`
	_, err := DB.Exec(query)
	return err
}

// UpdateMetric обновляет или создает метрику в БД
func UpdateMetric(id string, mType string, delta *int64, value *float64) error {
	query := `
		INSERT INTO metrics (id, type, delta, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id, type) DO UPDATE
		SET delta = $3, value = $4`

	_, err := DB.Exec(query, id, mType, delta, value)
	return err
}

func GetMetric(id string, mType string) (*int64, *float64, error) {
	var delta sql.NullInt64
	var value sql.NullFloat64

	query := `SELECT delta, value FROM metrics WHERE id = $1 AND type = $2`
	err := DB.QueryRow(query, id, mType).Scan(&delta, &value)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("metric not found")
		}
		return nil, nil, err
	}

	var deltaPtr *int64
	var valuePtr *float64

	if delta.Valid {
		deltaPtr = &delta.Int64
	}
	if value.Valid {
		valuePtr = &value.Float64
	}

	return deltaPtr, valuePtr, nil
}

func PingDB() error {
	return DB.Ping()
}
