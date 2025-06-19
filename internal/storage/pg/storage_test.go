package storage

import (
	"errors"
	"testing"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/retry"
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
			err := retry.WithRetry(tt.op)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
