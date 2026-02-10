package helpers

import (
	"strconv"
)

// ParsePagination parses limit and offset from query parameters
// Returns limit (default 20) and offset (default 0)
// If values are invalid, defaults are returned
func ParsePagination(limitStr, offsetStr string) (int, int) {
	limit := 20
	offset := 0

	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Safety cap on limit
	if limit > 100 {
		limit = 100
	}

	return limit, offset
}
