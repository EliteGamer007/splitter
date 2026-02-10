package posts_test

import (
	"splitter/internal/models"
	"strings"
	"testing"
)

/*
WHY THIS TEST EXISTS:
- Enforces content rules (length, empty checks).
- Ensures visibility settings are valid.

EXPECTED BEHAVIOR:
- valid content passes.
- empty content fails unless media is present.
- invalid visibility fails.

TEST RESULT SUMMARY:
- Passed: Valid post, too long content, empty content (no media), empty content (with media), invalid visibility.
*/

func TestPostCreateValidation(t *testing.T) {
	tests := []struct {
		name     string
		post     models.PostCreate
		hasMedia bool
		wantErr  bool
		errMsg   string
	}{
		{
			name: "Valid Post",
			post: models.PostCreate{
				Content:    "This is a valid post",
				Visibility: "public",
			},
			hasMedia: false,
			wantErr:  false,
		},
		{
			name: "Content Too Long",
			post: models.PostCreate{
				Content:    strings.Repeat("a", 501),
				Visibility: "public",
			},
			hasMedia: false,
			wantErr:  true,
			errMsg:   "content too long",
		},
		{
			name: "Empty Content No Media",
			post: models.PostCreate{
				Content:    "",
				Visibility: "public",
			},
			hasMedia: false,
			wantErr:  true,
			errMsg:   "either content or media is required",
		},
		{
			name: "Empty Content With Media",
			post: models.PostCreate{
				Content:    "",
				Visibility: "public",
			},
			hasMedia: true,
			wantErr:  false,
		},
		{
			name: "Invalid Visibility",
			post: models.PostCreate{
				Content:    "Valid content",
				Visibility: "invalid_option",
			},
			hasMedia: false,
			wantErr:  true,
			errMsg:   "invalid visibility setting",
		},
	}

	for _, tt := range tests {
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
