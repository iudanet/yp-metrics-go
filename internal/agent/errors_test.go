package agent

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPError(t *testing.T) {
	err := &HTTPError{
		StatusCode: http.StatusNotFound,
		Status:     "404 Not Found",
	}

	assert.Equal(t, "HTTP error: 404 Not Found", err.Error())
	assert.Equal(t, http.StatusNotFound, err.HTTPStatusCode())
}
