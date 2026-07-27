package setup

import "testing"

func TestReviewScrollTopForCursor(t *testing.T) {
	tests := []struct {
		name      string
		cursor    int
		scrollTop int
		total     int
		visible   int
		want      int
	}{
		{name: "empty list", cursor: 0, scrollTop: 0, total: 0, visible: 5, want: 0},
		{name: "cursor above window", cursor: 2, scrollTop: 5, total: 20, visible: 5, want: 2},
		{name: "cursor below window", cursor: 12, scrollTop: 0, total: 20, visible: 5, want: 8},
		{name: "cursor inside window", cursor: 3, scrollTop: 2, total: 20, visible: 5, want: 2},
		{name: "clamp to end", cursor: 19, scrollTop: 0, total: 20, visible: 5, want: 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reviewScrollTopForCursor(tt.cursor, tt.scrollTop, tt.total, tt.visible)
			if got != tt.want {
				t.Fatalf("reviewScrollTopForCursor() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestReviewListVisibleRows(t *testing.T) {
	if got := reviewListVisibleRows(24, true); got < 3 {
		t.Fatalf("visible rows too small: %d", got)
	}
	if got := reviewListVisibleRows(0, false); got != 10 {
		t.Fatalf("default visible rows = %d, want 10", got)
	}
}
