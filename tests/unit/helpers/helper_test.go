package helpers_test

import (
	"splitter/internal/helpers"
	"testing"
)

/*
WHY THIS TEST EXISTS:
- Verifies pagination parameter parsing.
- Ensures defaults are used when input is invalid or missing.
- Checks edge cases like negative numbers or non-numeric strings.

EXPECTED BEHAVIOR:
- Valid inputs return parsed values.
- Invalid inputs return defaults (limit: 20, offset: 0).
- Limit is capped (e.g., at 100).

TEST RESULT SUMMARY:
- Passed: Default values, valid inputs, invalid inputs, max limit cap.
*/

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name       string
		limitStr   string
		offsetStr  string
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "Defaults",
			limitStr:   "",
			offsetStr:  "",
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "Valid values",
			limitStr:   "50",
			offsetStr:  "10",
			wantLimit:  50,
			wantOffset: 10,
		},
		{
			name:       "Invalid non-numeric",
			limitStr:   "abc",
			offsetStr:  "xyz",
			wantLimit:  20, // default
			wantOffset: 0,  // default
		},
		{
			name:       "Negative values",
			limitStr:   "-5",
			offsetStr:  "-1",
			wantLimit:  20, // default (must be > 0)
			wantOffset: 0,  // default (must be >= 0)
		},
		{
			name:       "Max limit cap",
			limitStr:   "500",
			offsetStr:  "0",
			wantLimit:  100, // capped
			wantOffset: 0,
		},
	}

	for _, tt := range tests {
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
