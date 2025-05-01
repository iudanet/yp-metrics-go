package storage

import (
	"errors"
	"sync"

	"github.com/iudanet/yp-metrics-go/internal/utils"
)

var (
	ErrNotFound = errors.New("not found")
)

type Repository interface {
	SetCounter(string, int64) error
	IncrCounter(string) error
	SetGauge(string, float64) error
	GetCounter(name string) (int64, error)
	GetGauge(name string) (float64, error)
	GetMapGauge() (map[string]float64, error)
	GetMapCounter() (map[string]int64, error)
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
	m.gauge[name] = utils.Round(value, 3)
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
	return m.gauge, nil
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
