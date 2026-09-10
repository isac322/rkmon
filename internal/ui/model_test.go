package ui

import (
	"testing"

	"github.com/isac322/rkmon/internal/collect"
)

func TestMaxPerformanceToggleDoesNotStartAnotherRefreshLoop(t *testing.T) {
	model := Model{
		snap: &collect.Snapshot{},
		toggleMaxPerformance: func() (bool, error) {
			return true, nil
		},
	}

	updated, cmd := model.Update(keyPress('p'))
	if cmd != nil {
		t.Fatal("max-performance toggle returned a command; this starts an additional refresh loop")
	}
	got := updated.(Model)
	if !got.snap.Host.MaxPerformance {
		t.Fatal("successful toggle did not update the current snapshot")
	}
}
