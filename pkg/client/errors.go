package client

import (
	"errors"
	"fmt"
)

// HTTPError represents a non-2xx HTTP response from the API.
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// IsStatus returns true if err (or any wrapped error) is an HTTPError with the given status code.
func IsStatus(err error, code int) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == code
	}
	return false
}
