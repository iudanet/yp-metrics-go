package storage

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"sync"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"go.uber.org/zap"
)

var (
	ErrNotFound = errors.New("not found")
)

// MetricsDB хранилище для персистентного хранения метрик
type MetricsDB struct {
	Gauges   map[string]float64 `json:"gauges"`
	Counters map[string]int64   `json:"counters"`
}

// MetricReader определяет методы для чтения метрик
type MetricReader interface {
	GetCounter(name string) (int64, error)
	GetGauge(name string) (float64, error)
	GetMapGauge() (map[string]float64, error)
	GetMapCounter() (map[string]int64, error)
}

// MetricWriter определяет методы для записи метрик
type MetricWriter interface {
	SetCounter(string, int64) error
	SetGauge(string, float64) error
	MetricPersistent
}

type MetricPersistent interface {
	SaveDB(string) error
	LoadDB(string) error
}

// CounterIncrementer выделяет специфическую операцию инкремента
type CounterIncrementer interface {
	IncrCounter(string) error
}

// Repository объединяет все интерфейсы, если нужен полный функционал
type Repository interface {
	MetricReader
	MetricWriter
	CounterIncrementer
}

func NewStorage() *memStorage {
	return &memStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

type memStorage struct {
	gauge   map[string]float64
	counter map[string]int64
	mutex   sync.RWMutex
	wg      sync.WaitGroup
}

func (m *memStorage) SetCounter(name string, value int64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.counter[name]
	if ok {
		m.counter[name] = v + value
	} else {
		m.counter[name] = value
	}

	return nil
}

func (m *memStorage) SetGauge(name string, value float64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.gauge[name] = value

	return nil
}

func (m *memStorage) IncrCounter(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.counter[name]++
	return nil
}

func (m *memStorage) GetMapCounter() (map[string]int64, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.counter, nil
}

func (m *memStorage) GetMapGauge() (map[string]float64, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	// return map возвращает ссылку. надо ккопировать map или обрабатывать range через мютекс
	copyMapGauge := make(map[string]float64)
	maps.Copy(copyMapGauge, m.gauge)
	return copyMapGauge, nil
}

func (m *memStorage) GetCounter(name string) (int64, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	value, ok := m.counter[name]
	if !ok {
		return 0, ErrNotFound
	}
	return value, nil
}

func (m *memStorage) GetGauge(name string) (float64, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	value, ok := m.gauge[name]

	if !ok {
		return 0, ErrNotFound
	}
	return value, nil
}

func (m *memStorage) SaveDB(filename string) error {
	gauges, _ := m.GetMapGauge()
	counters, _ := m.GetMapCounter()
	db := MetricsDB{
		Gauges:   gauges,
		Counters: counters,
	}
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (m *memStorage) LoadDB(filename string) error {
	var db MetricsDB
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &db)
	if err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.counter = db.Counters
	m.gauge = db.Gauges

	return nil
}

func (m *memStorage) WaitWorker() {
	m.wg.Wait()
}

func (m *memStorage) StartWorker(ctx context.Context, cfg config.Storage, logger *zap.Logger) {
	// Используем StoreInterval из конфигурации
	interval := time.Duration(cfg.StoreInterval) * time.Second
	ticker := time.NewTicker(interval)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := m.SaveDB(cfg.Path)
				if err != nil {
					logger.Error("Failed to save metrics", zap.Error(err))
				}
				logger.Info("Auto Save complite")
			case <-ctx.Done():
				// Финальное сохранение при завершении

				err := m.SaveDB(cfg.Path)

				if err != nil {
					logger.Error("Failed to save metrics during shutdown", zap.Error(err))
				}
				logger.Info("Storage gracefully stopped")
				return
			}
		}
	}()
}
