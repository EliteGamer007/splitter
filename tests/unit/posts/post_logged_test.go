// Package posts_test — logger bridge for unit tests.
package posts_test

import (
	"splitter/internal/models"
	"strings"
	"testing"
	"time"

	"splitter/tests/testlogger"
)

func TestPostCreateValidationLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "unit", start, testErr) }()

	tests := []struct {
		name     string
		post     models.PostCreate
		hasMedia bool
		wantErr  bool
		errMsg   string
	}{
		{"Valid Post", models.PostCreate{Content: "Valid post", Visibility: "public"}, false, false, ""},
		{"Content Too Long", models.PostCreate{Content: strings.Repeat("a", 501), Visibility: "public"}, false, true, "content too long"},
		{"Empty Content No Media", models.PostCreate{Content: "", Visibility: "public"}, false, true, "either content or media is required"},
		{"Empty Content With Media", models.PostCreate{Content: "", Visibility: "public"}, true, false, ""},
		{"Invalid Visibility", models.PostCreate{Content: "Valid content", Visibility: "invalid"}, false, true, "invalid visibility"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.post.Validate(tt.hasMedia)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, expected to contain %v", err, tt.errMsg)
			}
		})
	}
}
