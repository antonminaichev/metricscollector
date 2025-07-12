package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/antonminaichev/metricscollector/internal/retry"
	"github.com/antonminaichev/metricscollector/internal/server/storage"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresStorage realieses storage interface for postgresDB.
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage creates new PostgreSQL storage.
func NewPostgresStorage(connStr string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	storage := &PostgresStorage{db: db}

	if err := storage.Ping(context.Background()); err != nil {
		return nil, err
	}

	if err := storage.initTable(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *PostgresStorage) initTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS metrics (
			id VARCHAR NOT NULL,
			type VARCHAR NOT NULL,
			delta BIGINT,
			value DOUBLE PRECISION,
			PRIMARY KEY (id, type)
		)`

	return retry.Do(retry.DefaultRetryConfig(), func() error {
		_, err := s.db.Exec(query)
		return err
	})
}

// UpdateMetric creates or updates metric in a DB storage.
func (s *PostgresStorage) UpdateMetric(ctx context.Context, id string, mType storage.MetricType, delta *int64, value *float64) error {
	query := `
		INSERT INTO metrics (id, type, delta, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id, type) DO UPDATE
		SET delta = $3 + metrics.delta, value = $4`

	return retry.Do(retry.DefaultRetryConfig(), func() error {
		_, err := s.db.ExecContext(ctx, query, id, string(mType), delta, value)
		return err
	})
}

// GetMetric returns a metric from a DB storage.
func (s *PostgresStorage) GetMetric(ctx context.Context, id string, mType storage.MetricType) (*int64, *float64, error) {
	var delta sql.NullInt64
	var value sql.NullFloat64

	query := `SELECT delta, value FROM metrics WHERE id = $1 AND type = $2`

	err := retry.Do(retry.DefaultRetryConfig(), func() error {
		return s.db.QueryRowContext(ctx, query, id, string(mType)).Scan(&delta, &value)
	})

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

// GetAllMetrics returns all existing metrics from a DB storage.
func (s *PostgresStorage) GetAllMetrics(ctx context.Context) (map[string]int64, map[string]float64, error) {
	counters := make(map[string]int64)
	gauges := make(map[string]float64)

	query := `SELECT id, type, delta, value FROM metrics`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var mType string
		var delta sql.NullInt64
		var value sql.NullFloat64

		if err := rows.Scan(&id, &mType, &delta, &value); err != nil {
			return nil, nil, err
		}

		if mType == string(storage.Counter) && delta.Valid {
			counters[id] = delta.Int64
		} else if mType == string(storage.Gauge) && value.Valid {
			gauges[id] = value.Float64
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return counters, gauges, nil
}

// Ping pings a DB for a availability.
func (s *PostgresStorage) Ping(ctx context.Context) error {
	return retry.Do(retry.DefaultRetryConfig(), func() error {
		return s.db.PingContext(ctx)
	})
}
