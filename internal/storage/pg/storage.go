package storage

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"github.com/jackc/pgx/v5"
)

var _ storage.Repository = (*postgreStorage)(nil)

var (
	ErrNotFound = errors.New("not found")
)

type postgreStorage struct {
	conn *pgx.Conn
}

func New(ctx context.Context, dsn string) *postgreStorage {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
		os.Exit(1)
	}

	// Создаем таблицы если их нет
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS gauges (
			name TEXT PRIMARY KEY,
			value DOUBLE PRECISION NOT NULL
		);

		CREATE TABLE IF NOT EXISTS counters (
			name TEXT PRIMARY KEY,
			value BIGINT NOT NULL
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
		os.Exit(1)
	}

	return &postgreStorage{
		conn: conn,
	}
}

func (p *postgreStorage) SetCounter(ctx context.Context, name string, value int64) error {
	_, err := p.conn.Exec(ctx, `
		INSERT INTO counters (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE
		SET value = counters.value + EXCLUDED.value
	`, name, value)
	return err
}

func (p *postgreStorage) SetGauge(ctx context.Context, name string, value float64) error {
	_, err := p.conn.Exec(ctx, `
		INSERT INTO gauges (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE
		SET value = EXCLUDED.value
	`, name, value)
	return err
}

func (p *postgreStorage) IncrCounter(ctx context.Context, name string) error {
	_, err := p.conn.Exec(ctx, `
		INSERT INTO counters (name, value)
		VALUES ($1, 1)
		ON CONFLICT (name) DO UPDATE
		SET value = counters.value + 1
	`, name)
	return err
}

func (p *postgreStorage) GetMapCounter(ctx context.Context) (map[string]int64, error) {
	rows, err := p.conn.Query(ctx, "SELECT name, value FROM counters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result[name] = value
	}
	return result, nil
}

func (p *postgreStorage) GetMapGauge(ctx context.Context) (map[string]float64, error) {
	rows, err := p.conn.Query(ctx, "SELECT name, value FROM gauges")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result[name] = value
	}
	return result, nil
}

func (p *postgreStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	var value int64
	err := p.conn.QueryRow(ctx,
		"SELECT value FROM counters WHERE name = $1", name).Scan(&value)
	if err == pgx.ErrNoRows {
		return 0, ErrNotFound
	}
	return value, err
}

func (p *postgreStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	var value float64
	err := p.conn.QueryRow(ctx,
		"SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
	if err == pgx.ErrNoRows {
		return 0, ErrNotFound
	}
	return value, err
}

func (p *postgreStorage) SaveDB(ctx context.Context, filename string) error {
	return nil
}

func (p *postgreStorage) LoadDB(ctx context.Context, filename string) error {
	return nil
}

func (p *postgreStorage) Ping(ctx context.Context) error {
	err := p.conn.Ping(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *postgreStorage) WriteBatch(ctx context.Context, metrics []models.Metrics) error {
	tx, err := p.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, metric := range metrics {
		switch metric.MType {
		case "counter":
			_, err := tx.Exec(ctx, `
				INSERT INTO counters (name, value)
				VALUES ($1, $2)
				ON CONFLICT (name) DO UPDATE
				SET value = counters.value + EXCLUDED.value
			`, metric.ID, *metric.Delta)
			if err != nil {
				return err
			}
		case "gauge":
			_, err := tx.Exec(ctx, `
				INSERT INTO gauges (name, value)
				VALUES ($1, $2)
				ON CONFLICT (name) DO UPDATE
				SET value = EXCLUDED.value
			`, metric.ID, *metric.Value)
			if err != nil {
				return err
			}
		default:
			return errors.New("invalid metric type")
		}
	}
	return tx.Commit(ctx)
}
