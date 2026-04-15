package quota

import (
	"path/filepath"
	"testing"
)

func TestTakeDailyLimit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "q")
	if err := TakeDaily(dir, 2); err != nil {
		t.Fatal(err)
	}
	if err := TakeDaily(dir, 2); err != nil {
		t.Fatal(err)
	}
	if err := TakeDaily(dir, 2); err == nil {
		t.Fatal("expected limit error")
	}
}

func TestTakeDailyUnlimited(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "q2")
	for i := 0; i < 5; i++ {
		if err := TakeDaily(dir, 0); err != nil {
			t.Fatal(err)
		}
	}
}
