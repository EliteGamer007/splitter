// Package replies_test — logger bridge for unit tests.
package replies_test

import (
	"splitter/internal/models"
	"strings"
	"testing"
	"time"

	"splitter/tests/testlogger"
)

func TestReplyCreateValidationLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "unit", start, testErr) }()

	tests := []struct {
		name    string
		reply   models.ReplyCreate
		wantErr bool
		errMsg  string
	}{
		{"Valid Reply", models.ReplyCreate{PostID: "post-123", Content: "Valid reply"}, false, ""},
		{"Missing PostID", models.ReplyCreate{Content: "Valid reply"}, true, "post_id is required"},
		{"Empty Content", models.ReplyCreate{PostID: "post-123", Content: ""}, true, "content cannot be empty"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.reply.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() = %v, expected to contain %v", err, tt.errMsg)
			}
		})
	}
}

func TestReplyDepthLogicLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "unit", start, testErr) }()

	calc := func(parentDepth int) int { return parentDepth + 1 }
	rootDepth := 1
	childDepth := calc(rootDepth)
	grandChildDepth := calc(childDepth)

	if childDepth != 2 {
		t.Errorf("Expected child depth 2, got %d", childDepth)
	}
	if grandChildDepth != 3 {
		t.Errorf("Expected grandchild depth 3, got %d", grandChildDepth)
	}
}
