package api

import (
	"testing"
)

func TestFindFilterConstruction(t *testing.T) {
	f := &FindFilter{}
	if f == nil {
		t.Fatal("FindFilter construction returned nil")
	}
}

func TestNoTagsConstant(t *testing.T) {
	if NoTags != "no tags" {
		t.Errorf("NoTags = %q, want %q", NoTags, "no tags")
	}
}
