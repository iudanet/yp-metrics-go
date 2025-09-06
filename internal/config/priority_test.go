package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

func TestParseJSONConfigFile(t *testing.T) {
	// Тест ошибки чтения
	er := &errorReader{}
	err := parseJSONConfigFile(er, &rawAgentConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read config file error")

	// Тест ошибки json
	data := strings.NewReader("invalid json")
	err = parseJSONConfigFile(data, &rawAgentConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal config json error")
}
