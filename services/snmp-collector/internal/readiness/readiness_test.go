package readiness

import "testing"

func TestEvaluate(t *testing.T) {
	t.Parallel()

	if !Evaluate(true, true, true, true).Ready() {
		t.Fatal("expected ready")
	}
	if Evaluate(true, true, true, false).Ready() {
		t.Fatal("publisher down should not be ready")
	}
	if Evaluate(false, true, true, true).Ready() {
		t.Fatal("missing config should not be ready")
	}
}
