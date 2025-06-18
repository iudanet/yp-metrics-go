package storage

import (
	"context"
	"testing"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgStorage(t *testing.T) {
	dsn := "postgres://postgres:postgres@localhost:5432/metrics_db?sslmode=disable"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Пробуем подключиться
	store, err := New(ctx, dsn)
	if err != nil {
		t.Skipf("Skipping test due to database connection error: %v", err)
	}
	require.NoError(t, err)

	// Очищаем таблицы перед тестом
	_, err = store.pool.Exec(ctx, "TRUNCATE TABLE gauges, counters")
	require.NoError(t, err)

	t.Run("Set and get gauge", func(t *testing.T) {
		err := store.SetGauge(ctx, "testGauge", 10.5)
		assert.NoError(t, err)

		value, err := store.GetGauge(ctx, "testGauge")
		assert.NoError(t, err)
		assert.Equal(t, 10.5, value)
	})

	t.Run("Set and get counter", func(t *testing.T) {
		err := store.SetCounter(ctx, "testCounter", 5)
		assert.NoError(t, err)

		value, err := store.GetCounter(ctx, "testCounter")
		assert.NoError(t, err)
		assert.Equal(t, int64(5), value)
	})

	t.Run("Increment counter", func(t *testing.T) {
		err := store.IncrCounter(ctx, "testCounter")
		assert.NoError(t, err)

		value, err := store.GetCounter(ctx, "testCounter")
		assert.NoError(t, err)
		assert.Equal(t, int64(6), value)
	})

	t.Run("Get non-existent metric", func(t *testing.T) {
		_, err := store.GetGauge(ctx, "unknown")
		assert.ErrorIs(t, err, ErrNotFound)

		_, err = store.GetCounter(ctx, "unknown")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Write batch", func(t *testing.T) {
		metrics := []models.Metrics{
			{ID: "batchGauge", MType: "gauge", Value: ptrFloat64(12.3)},
			{ID: "batchCounter", MType: "counter", Delta: ptrInt64(7)},
		}

		err := store.WriteBatch(ctx, metrics)
		assert.NoError(t, err)

		gauge, err := store.GetGauge(ctx, "batchGauge")
		assert.NoError(t, err)
		assert.Equal(t, 12.3, gauge)

		counter, err := store.GetCounter(ctx, "batchCounter")
		assert.NoError(t, err)
		assert.Equal(t, int64(7), counter)
	})
}

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt64(v int64) *int64       { return &v }
