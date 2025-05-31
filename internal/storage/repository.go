package storage

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("not found")
)

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
type HealthcheckDB interface {
	Ping(context.Context) error
}

// Repository объединяет все интерфейсы, если нужен полный функционал
type Repository interface {
	MetricReader
	MetricWriter
	CounterIncrementer
	HealthcheckDB
}
