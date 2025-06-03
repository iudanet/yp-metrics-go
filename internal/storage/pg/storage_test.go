package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestWithRetry(t *testing.T) {
	tests := []struct {
		name      string
		op        func() error
		wantError bool
	}{
		{
			name: "success on first try",
			op: func() error {
				return nil
			},
			wantError: false,
		},
		{
			name: "retriable error then success",
			op: func() error {
				if time.Now().Second()%2 == 0 { // simple way to make it fail first time
					return &pgconn.PgError{Code: pgerrcode.ConnectionException}
				}
				return nil
			},
			wantError: false,
		},
		{
			name: "non-retriable error",
			op: func() error {
				return errors.New("non-retriable error")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := withRetry(tt.op)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

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
			name:     "context canceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "non-retriable error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isRetriableError(tt.err))
		})
	}
}
