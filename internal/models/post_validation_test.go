package models

import "testing"

func TestCirclePostsAreValid(t *testing.T) {
	p := PostCreate{Content: "circle post", Visibility: "circle"}
	if err := p.Validate(false); err != nil {
		t.Fatal(err)
	}
}
