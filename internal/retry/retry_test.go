package retry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

type mockHTTPError struct {
	statusCode int
}

func (m *mockHTTPError) HTTPStatusCode() int {
	return m.statusCode
}

func (m *mockHTTPError) Error() string {
	return fmt.Sprintf("HTTP %d error", m.statusCode)
}

func TestIsRetriableError(t *testing.T) {
	tests := []struct {
		err      error
		name     string
		expected bool
	}{
		{
			name:     "HTTP500Error",
			err:      &mockHTTPError{statusCode: 500},
			expected: true,
		},
		{
			name:     "HTTP400Error",
			err:      &mockHTTPError{statusCode: 400},
			expected: false,
		},
		{
			name:     "pgConnectionError",
			err:      &pgconn.PgError{Code: pgerrcode.ConnectionException},
			expected: true,
		},
		{
			name:     "networkError",
			err:      &net.DNSError{IsTimeout: true},
			expected: true,
		},
		{
			name:     "contextCanceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "closedNetworkConnection",
			err:      errors.New("use of closed network connection"),
			expected: true,
		},
		{
			name:     "backendError",
			err:      errors.New("database is not available"),
			expected: false,
		},
		{
			name:     "non-retriablError",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "retriedConnectionException",
			err:      &pgconn.PgError{Code: pgerrcode.TransactionResolutionUnknown},
			expected: true,
		},
		{
			name:     "retriedConnectionDoesNotExist",
			err:      &pgconn.PgError{Code: pgerrcode.ConnectionDoesNotExist},
			expected: true,
		},
		{
			name:     "retried_SQL_client_unable_to_establish_connection",
			err:      &pgconn.PgError{Code: pgerrcode.SQLClientUnableToEstablishSQLConnection},
			expected: true,
		},
		{
			name:     "invalid_connection_error",
			err:      errors.New("invalid connection"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetriable(tt.err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
