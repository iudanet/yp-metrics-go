package storage

import (
	"errors"
	"sync"

	"github.com/iudanet/yp-metrics-go/internal/utils"
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

func NewStorage() Repository {
	return &memStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

var (
	ErrNotFound = errors.New("not found")
)

type memStorage struct {
	gauge   map[string]float64
	counter map[string]int64
	mutex   sync.RWMutex
}

func (m *memStorage) SetCounter(name string, value int64) error {
	defer m.mutex.Unlock()
	m.mutex.Lock()
	v, ok := m.counter[name]
	if ok {
		m.counter[name] = v + value
	} else {
		m.counter[name] = value
	}

	return nil
}

func (m *memStorage) SetGauge(name string, value float64) error {
	defer m.mutex.Unlock()
	m.mutex.Lock()
	m.gauge[name] = utils.Round(value, 3)
	return nil
}

func (m *memStorage) IncrCounter(name string) error {
	defer m.mutex.Unlock()
	m.mutex.Lock()
	m.counter[name]++
	return nil
}

func (m *memStorage) GetMapCounter() (map[string]int64, error) {
	defer m.mutex.RUnlock()
	m.mutex.RLock()
	return m.counter, nil
}

func (m *memStorage) GetMapGauge() (map[string]float64, error) {
	defer m.mutex.RUnlock()
	m.mutex.RLock()
	return m.gauge, nil
}

func (m *memStorage) GetCounter(name string) (int64, error) {
	defer m.mutex.RUnlock()
	m.mutex.RLock()
	value, ok := m.counter[name]
	if !ok {
		return 0, ErrNotFound
	}
	return value, nil
}

func (m *memStorage) GetGauge(name string) (float64, error) {
	defer m.mutex.RUnlock()
	m.mutex.RLock()
	value, ok := m.gauge[name]

	if !ok {
		return 0, ErrNotFound
	}
	return value, nil
}
