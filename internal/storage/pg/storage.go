package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/retry"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ storage.Repository = (*postgreStorage)(nil)

const (
	pgPoolMaxConns          = 10
	pgPoolMinConns          = 2
	pgPollMaxConnLifetime   = 30 * time.Minute
	pgPollMinConnIdleTime   = 5 * time.Minute
	pgPollHealthCheckPeriod = 1 * time.Minute
)

var (
	ErrNotFound = errors.New("not found")
)

type postgreStorage struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*postgreStorage, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgxpool config: %w", err)
	}

	// Настраиваем пул соединений
	config.MaxConns = pgPoolMaxConns
	config.MinConns = pgPoolMinConns
	config.MaxConnLifetime = pgPollMaxConnLifetime
	config.MaxConnIdleTime = pgPollMinConnIdleTime
	config.HealthCheckPeriod = pgPollHealthCheckPeriod

	var pool *pgxpool.Pool

	// Добавляем retry при подключении
	err = retry.WithRetry(func() error {
		var err error
		pool, err = pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			return err
		}
		return pool.Ping(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	// Создаем таблицы если их нет
	err = retry.WithRetry(func() error {
		conn, err := pool.Acquire(ctx)
		if err != nil {
			return err
		}
		defer conn.Release()

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
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &postgreStorage{
		pool: pool,
	}, nil
}

func (p *postgreStorage) SetCounter(ctx context.Context, name string, value int64) error {
	return retry.WithRetry(func() error {
		conn, err := p.pool.Acquire(ctx)
		if err != nil {
			return err
		}
		defer conn.Release()

		_, err = conn.Exec(ctx, `
			INSERT INTO counters (name, value)
			VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE
			SET value = counters.value + EXCLUDED.value
		`, name, value)
		return err
	})
}

func (p *postgreStorage) IncrCounter(ctx context.Context, name string) error {
	return retry.WithRetry(func() error {
		conn, err := p.pool.Acquire(ctx)
		if err != nil {
			return err
		}
		defer conn.Release()

		_, err = conn.Exec(ctx, `
		INSERT INTO counters (name, value)
		VALUES ($1, 1)
		ON CONFLICT (name) DO UPDATE
		SET value = counters.value + 1
	`, name)
		return err
	})
}

func (p *postgreStorage) SetGauge(ctx context.Context, name string, value float64) error {
	return retry.WithRetry(func() error {
		conn, err := p.pool.Acquire(ctx)
		if err != nil {
			return err
		}
		defer conn.Release()

		_, err = conn.Exec(ctx, `
			INSERT INTO gauges (name, value)
			VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE
			SET value = EXCLUDED.value
		`, name, value)
		return err
	})
}

func (p *postgreStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	var value int64
	err := retry.WithRetry(func() error {
		conn, err := p.pool.Acquire(ctx)
		if err != nil {
			return err
		}
		defer conn.Release()

		err = conn.QueryRow(ctx,
			"SELECT value FROM counters WHERE name = $1", name).Scan(&value)
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return err
	})
	return value, err
}

func (p *postgreStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	var value float64
	err := retry.WithRetry(func() error {
		conn, err := p.pool.Acquire(ctx)
		if err != nil {
			return err
		}
		defer conn.Release()

		err = conn.QueryRow(ctx,
			"SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return err
	})
	return value, err
}

func (p *postgreStorage) GetMapCounter(ctx context.Context) (map[string]int64, error) {
	var result map[string]int64
	err := retry.WithRetry(func() error {
		conn, err := p.pool.Acquire(ctx)
		if err != nil {
			return err
		}
		defer conn.Release()

		rows, err := conn.Query(ctx, "SELECT name, value FROM counters")
		if err != nil {
			return err
		}
		defer rows.Close()

		result = make(map[string]int64)
		for rows.Next() {
			var name string
			var value int64
			if err := rows.Scan(&name, &value); err != nil {
				return err
			}
			result[name] = value
		}
		return nil
	})
	return result, err
}

func (p *postgreStorage) GetMapGauge(ctx context.Context) (map[string]float64, error) {
	var result map[string]float64
	err := retry.WithRetry(func() error {
		conn, err := p.pool.Acquire(ctx)
		if err != nil {
			return err
		}
		defer conn.Release()

		rows, err := conn.Query(ctx, "SELECT name, value FROM gauges")
		if err != nil {
			return err
		}
		defer rows.Close()

		result = make(map[string]float64)
		for rows.Next() {
			var name string
			var value float64
			if err := rows.Scan(&name, &value); err != nil {
				return err
			}
			result[name] = value
		}
		return nil
	})
	return result, err
}

func (p *postgreStorage) Ping(ctx context.Context) error {
	return retry.WithRetry(func() error {
		return p.pool.Ping(ctx)
	})
}

func (p *postgreStorage) WriteBatch(ctx context.Context, metrics []models.Metrics) error {
	return retry.WithRetry(func() error {
		conn, err := p.pool.Acquire(ctx)
		if err != nil {
			return err
		}
		defer conn.Release()

		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		for _, metric := range metrics {
			switch metric.MType {
			case models.TypeCounter:
				_, err := tx.Exec(ctx, `
					INSERT INTO counters (name, value)
					VALUES ($1, $2)
					ON CONFLICT (name) DO UPDATE
					SET value = counters.value + EXCLUDED.value
				`, metric.ID, *metric.Delta)
				if err != nil {
					return err
				}
			case models.TypeGauge:
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
	})
}

func (p *postgreStorage) LoadDB(ctx context.Context, filename string) error {
	return nil
}
func (p *postgreStorage) SaveDB(ctx context.Context, filename string) error {
	return nil
}
