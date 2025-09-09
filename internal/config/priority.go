package config

import (
	"encoding/json"
	"fmt"
	"io"
)

// приоритеты, 1 - наивысший (env), 4 - самый низкий (дефолт)
const (
	priorityEnv        = 1
	priorityFlags      = 2
	priorityConfigFile = 3
	priorityDefault    = 4
)

type prioritized[T any] struct {
	value    T
	priority int
}

func (p *prioritized[T]) Set(value T, priority int) {
	if p.priority == 0 || priority < p.priority {
		p.value = value
		p.priority = priority
	}
}

func (p *prioritized[T]) Get() T {
	return p.value
}

func parseJSONConfigFile(file io.Reader, cfg any) error {
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read config file error: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("unmarshal config json error: %w", err)
	}
	return nil
}
