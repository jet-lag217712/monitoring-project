package handlers

import (
	"strings"
	"testing"
)

func TestParsePatchSiteLocation(t *testing.T) {
	t.Parallel()

	t.Run("sets trimmed location", func(t *testing.T) {
		loc, err := parsePatchSiteLocation(strings.NewReader(`{"location":"  District Office  "}`))
		if err != nil {
			t.Fatal(err)
		}
		if loc == nil || *loc != "District Office" {
			t.Fatalf("location=%v", loc)
		}
	})

	t.Run("clears on empty", func(t *testing.T) {
		loc, err := parsePatchSiteLocation(strings.NewReader(`{"location":"   "}`))
		if err != nil {
			t.Fatal(err)
		}
		if loc != nil {
			t.Fatalf("expected nil clear, got %v", *loc)
		}
	})

	t.Run("requires location field", func(t *testing.T) {
		_, err := parsePatchSiteLocation(strings.NewReader(`{}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		_, err := parsePatchSiteLocation(strings.NewReader(`{"location":"A","extra":true}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects overlong location", func(t *testing.T) {
		long := strings.Repeat("a", maxSiteLocationLen+1)
		_, err := parsePatchSiteLocation(strings.NewReader(`{"location":"` + long + `"}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
