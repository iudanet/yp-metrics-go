// Package retry provides functionality for retrying operations that may fail temporarily.
// It includes mechanisms for automatic retries with backoff for network and database operations.
package retry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrMaxRetriesReached = errors.New("maximum retries reached")

// withRetry выполняет операцию с повторными попытками при retriable ошибках
func WithRetry(op func() error) error {
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

		if !isRetriable(err) {
			return err
		}

		lastErr = err
		log.Printf("retriable error occurred: %v, retrying in %v", err, backoff)
		time.Sleep(backoff)

	}

	return fmt.Errorf("%w: %v", ErrMaxRetriesReached, lastErr)
}

// isRetriable проверяет, является ли ошибка retriable
func isRetriable(err error) bool {
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
	// Ошибки HTTP 5xx считаем retriable
	var httpErr interface {
		HTTPStatusCode() int
	}
	if errors.As(err, &httpErr) && httpErr.HTTPStatusCode() >= 500 {
		return true
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
