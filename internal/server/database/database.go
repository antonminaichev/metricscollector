package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type DB struct {
	conn   *pgx.Conn
	logger *zap.Logger
	dsn    string
}

func NewDBConnection(dsn string, logger *zap.Logger) (*DB, error) {
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return &DB{
		conn:   conn,
		logger: logger,
		dsn:    dsn,
	}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	//  return db.conn.Ping(ctx)
	if err := db.conn.Ping(ctx); err != nil {
		db.logger.Info("Attempting to reconnect to database")

		// Закрываем старое соединение
		if db.conn != nil {
			db.conn.Close(ctx)
		}

		// Пытаемся переподключиться
		conn, err := pgx.Connect(ctx, db.dsn)
		if err != nil {
			return fmt.Errorf("error reconnecting to database: %w", err)
		}

		db.conn = conn
		return nil
	}
	return nil
}

func (db *DB) Close() {
	if db.conn != nil {
		db.conn.Close(context.Background())
	}
}
