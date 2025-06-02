package storage

import (
	"context"
)

// MetricReader определяет методы для чтения метрик
type MetricReader interface {
	GetCounter(ctx context.Context, name string) (int64, error)
	GetGauge(ctx context.Context, name string) (float64, error)
	GetMapGauge(ctx context.Context) (map[string]float64, error)
	GetMapCounter(ctx context.Context) (map[string]int64, error)
}

// MetricWriter определяет методы для записи метрик
type MetricWriter interface {
	SetCounter(ctx context.Context, name string, value int64) error
	SetGauge(ctx context.Context, name string, value float64) error
	MetricPersistent
}

type MetricPersistent interface {
	SaveDB(ctx context.Context, name string) error
	LoadDB(ctx context.Context, name string) error
}

// CounterIncrementer выделяет специфическую операцию инкремента
type CounterIncrementer interface {
	IncrCounter(ctx context.Context, name string) error
}
type HealthcheckDB interface {
	Ping(ctx context.Context) error
}

// Repository объединяет все интерфейсы, если нужен полный функционал
type Repository interface {
	MetricReader
	MetricWriter
	CounterIncrementer
	HealthcheckDB
}
