package retry

import (
	"errors"
	"testing"

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
			name: "Success_on_first_attempt",
			op: func() error {
				return nil
			},
			wantError: false,
		},
		{
			name: "Retriable_error_then_success",
			op: func() func() error {
				attempt := 0
				return func() error {
					if attempt < 2 {
						attempt++
						return &pgconn.PgError{Code: pgerrcode.ConnectionException}
					}
					return nil
				}
			}(), // вызываем внешнюю функцию, чтобы вернуть func() error
			wantError: false,
		},
		{
			name: "Non-retriable error",
			op: func() error {
				return errors.New("non-retriable error")
			},
			wantError: true,
		},
		{
			name: "All_attempts_fail",
			op: func() error {
				return &pgconn.PgError{Code: pgerrcode.ConnectionException}
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WithRetry(tt.op)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
