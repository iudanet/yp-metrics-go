package storage

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

type postgreStorage struct {
	conn *pgx.Conn
}

func NewPostgres(ctx context.Context, dsn string) *postgreStorage {
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

func (p *postgreStorage) SetCounter(name string, value int64) error {
	_, err := p.conn.Exec(context.Background(), `
		INSERT INTO counters (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE
		SET value = counters.value + EXCLUDED.value
	`, name, value)
	return err
}

func (p *postgreStorage) SetGauge(name string, value float64) error {
	_, err := p.conn.Exec(context.Background(), `
		INSERT INTO gauges (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE
		SET value = EXCLUDED.value
	`, name, value)
	return err
}

func (p *postgreStorage) IncrCounter(name string) error {
	_, err := p.conn.Exec(context.Background(), `
		INSERT INTO counters (name, value)
		VALUES ($1, 1)
		ON CONFLICT (name) DO UPDATE
		SET value = counters.value + 1
	`, name)
	return err
}

func (p *postgreStorage) GetMapCounter() (map[string]int64, error) {
	rows, err := p.conn.Query(context.Background(), "SELECT name, value FROM counters")
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

func (p *postgreStorage) GetMapGauge() (map[string]float64, error) {
	rows, err := p.conn.Query(context.Background(), "SELECT name, value FROM gauges")
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

func (p *postgreStorage) GetCounter(name string) (int64, error) {
	var value int64
	err := p.conn.QueryRow(context.Background(),
		"SELECT value FROM counters WHERE name = $1", name).Scan(&value)
	if err == pgx.ErrNoRows {
		return 0, ErrNotFound
	}
	return value, err
}

func (p *postgreStorage) GetGauge(name string) (float64, error) {
	var value float64
	err := p.conn.QueryRow(context.Background(),
		"SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
	if err == pgx.ErrNoRows {
		return 0, ErrNotFound
	}
	return value, err
}

func (p *postgreStorage) SaveDB(filename string) error {
	return nil
}

func (p *postgreStorage) LoadDB(filename string) error {
	return nil
}

func (s *postgreStorage) Ping(ctx context.Context) error {
	err := s.conn.Ping(ctx)
	if err != nil {
		return err
	}
	return nil
}
