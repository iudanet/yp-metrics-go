package retry

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsRetriableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "pg connection error",
			err:      &pgconn.PgError{Code: pgerrcode.ConnectionException},
			expected: true,
		},
		{
			name:     "network error",
			err:      &net.DNSError{IsTimeout: true},
			expected: true,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "closed network connection",
			err:      errors.New("use of closed network connection"),
			expected: true,
		},
		{
			name:     "backend error",
			err:      errors.New("database is not available"),
			expected: false,
		},
		{
			name:     "non-retriable error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "retried connection exception",
			err:      &pgconn.PgError{Code: pgerrcode.TransactionResolutionUnknown},
			expected: true,
		},
		{
			name:     "retried connection does not exist",
			err:      &pgconn.PgError{Code: pgerrcode.ConnectionDoesNotExist},
			expected: true,
		},
		{
			name:     "retried SQL client unable to establish connection",
			err:      &pgconn.PgError{Code: pgerrcode.SQLClientUnableToEstablishSQLConnection},
			expected: true,
		},
		{
			name:     "invalid connection error",
			err:      errors.New("invalid connection"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetriableError(tt.err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
