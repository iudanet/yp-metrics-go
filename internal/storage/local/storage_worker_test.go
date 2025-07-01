package storage

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWorker(t *testing.T) {
	// Создаем временный файл
	tmpFile, err := os.CreateTemp("", "metrics-db-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	store := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := zap.NewDevelopment()

	cfg := config.Storage{
		Path:          tmpFile.Name(),
		StoreInterval: 1, // 1 секунда
		Restore:       false,
	}

	// Запускаем воркер
	wg := &sync.WaitGroup{}
	wg.Add(1)
	store.StartWorker(ctx, cfg, logger, wg)

	// Добавляем тестовые данные
	store.SetGauge(ctx, "testGauge", 10.5)
	store.SetCounter(ctx, "testCounter", 5)

	// Ждем достаточно времени для срабатывания автосохранения
	time.Sleep(1500 * time.Millisecond)

	// Проверяем, что файл был создан и содержит данные
	fileInfo, err := os.Stat(tmpFile.Name())
	assert.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))

	// Останавливаем воркер
	cancel()
	wg.Wait()

	// Проверяем, что данные сохранились
	newStore := New()
	err = newStore.LoadDB(ctx, tmpFile.Name())
	assert.NoError(t, err)

	gauges, err := newStore.GetMapGauge(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 10.5, gauges["testGauge"])

	counters, err := newStore.GetMapCounter(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), counters["testCounter"])
}
