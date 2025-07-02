package agent

import "fmt"

type HTTPError struct {
	StatusCode int
	Status     string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP error: %s", e.Status)
}

func (e *HTTPError) HTTPStatusCode() int {
	return e.StatusCode
}
