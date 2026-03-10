// Package helpers_test — logger bridge for unit tests.
package helpers_test

import (
	"splitter/internal/helpers"
	"testing"
	"time"

	"splitter/tests/testlogger"
)

func TestParsePaginationLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "unit", start, testErr) }()

	tests := []struct {
		name       string
		limitStr   string
		offsetStr  string
		wantLimit  int
		wantOffset int
	}{
		{"Defaults", "", "", 20, 0},
		{"Valid values", "50", "10", 50, 10},
		{"Invalid non-numeric", "abc", "xyz", 20, 0},
		{"Negative values", "-5", "-1", 20, 0},
		{"Max limit cap", "500", "0", 100, 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotOffset := helpers.ParsePagination(tt.limitStr, tt.offsetStr)
			if gotLimit != tt.wantLimit {
				t.Errorf("ParsePagination() limit = %v, want %v", gotLimit, tt.wantLimit)
			}
			if gotOffset != tt.wantOffset {
				t.Errorf("ParsePagination() offset = %v, want %v", gotOffset, tt.wantOffset)
			}
		})
	}
}
