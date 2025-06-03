package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ storage.Repository = (*postgreStorage)(nil)

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
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	var pool *pgxpool.Pool

	// Добавляем retry при подключении
	err = withRetry(func() error {
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
	err = withRetry(func() error {
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

// withRetry выполняет операцию с повторными попытками при retriable ошибках
func withRetry(op func() error) error {
	var lastErr error
	backoffSchedule := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}

	for _, backoff := range backoffSchedule {
		err := op()
		if err == nil {
			return nil
		}

		// Проверяем, является ли ошибка retriable
		if isRetriableError(err) {
			lastErr = err
			log.Printf("retriable error occurred: %v, retrying in %v", err, backoff)
			time.Sleep(backoff)
			continue
		}

		// Не retriable ошибка
		return err
	}

	return fmt.Errorf("failed after retries: %w", lastErr)
}

// isRetriableError проверяет, является ли ошибка retriable
func isRetriableError(err error) bool {
	// Проверяем ошибки PostgreSQL
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Ошибки соединения (Class 08)
		if pgErr.Code == pgerrcode.ConnectionException ||
			pgErr.Code == pgerrcode.ConnectionDoesNotExist ||
			pgErr.Code == pgerrcode.ConnectionFailure ||
			pgErr.Code == pgerrcode.SQLClientUnableToEstablishSQLConnection ||
			pgErr.Code == pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection ||
			pgErr.Code == pgerrcode.TransactionResolutionUnknown {
			return true
		}
	}

	// Проверяем сетевые ошибки
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Проверяем закрытые соединения
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}

	// Проверяем другие retriable ошибки
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, pgx.ErrTxClosed) {
		return true
	}

	return false
}

func (p *postgreStorage) SetCounter(ctx context.Context, name string, value int64) error {
	return withRetry(func() error {
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
	return withRetry(func() error {
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
	return withRetry(func() error {
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
	err := withRetry(func() error {
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
	err := withRetry(func() error {
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
	err := withRetry(func() error {
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
	err := withRetry(func() error {
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
	return withRetry(func() error {
		return p.pool.Ping(ctx)
	})
}

func (p *postgreStorage) WriteBatch(ctx context.Context, metrics []models.Metrics) error {
	return withRetry(func() error {
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
