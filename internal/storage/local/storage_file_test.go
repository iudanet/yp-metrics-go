package storage

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileOperations(t *testing.T) {
	t.Run("Save and load DB", func(t *testing.T) {
		// Создаем временный файл
		tmpFile, err := os.CreateTemp("", "metrics-db-*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		store := New()
		ctx := context.Background()

		// Добавляем тестовые данные
		store.SetGauge(ctx, "testGauge", 10.5)
		store.SetCounter(ctx, "testCounter", 5)

		// Сохраняем
		err = store.SaveDB(ctx, tmpFile.Name())
		assert.NoError(t, err)

		// Загружаем в новое хранилище
		newStore := New()
		err = newStore.LoadDB(ctx, tmpFile.Name())
		assert.NoError(t, err)

		// Проверяем данные
		gauges, err := newStore.GetMapGauge(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 10.5, gauges["testGauge"])

		counters, err := newStore.GetMapCounter(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), counters["testCounter"])
	})

	t.Run("Load non-existent file", func(t *testing.T) {
		store := New()
		err := store.LoadDB(context.Background(), "/path/to/non-existent/file.json")
		assert.Error(t, err)
	})

	t.Run("Get counter and gauge", func(t *testing.T) {
		store := New()
		ctx := context.Background()

		// Добавляем тестовые данные
		store.SetGauge(ctx, "testGauge", 10.5)
		store.SetCounter(ctx, "testCounter", 5)

		// Получаем значения
		gauge, err := store.GetGauge(ctx, "testGauge")
		assert.NoError(t, err)
		assert.Equal(t, 10.5, gauge)

		counter, err := store.GetCounter(ctx, "testCounter")
		assert.NoError(t, err)
		assert.Equal(t, int64(5), counter)

		// Несуществующие метрики
		_, err = store.GetGauge(ctx, "unknown")
		assert.Error(t, err)

		_, err = store.GetCounter(ctx, "unknown")
		assert.Error(t, err)
	})
}
