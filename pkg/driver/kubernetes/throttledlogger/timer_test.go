package throttledlogger

import (
	"testing"
	"time"
)

func TestTimer_TickAndIntervalPassed(t *testing.T) {
	interval := time.Millisecond * 100
	timer := NewTimer(interval)

	// Ensure initially the interval has not passed
	now := time.Now()
	if timer.IntervalPassed(now) {
		t.Fatalf("expected interval not passed yet, got interval passed")
	}

	// Move forward in time beyond the interval
	now = now.Add(interval + time.Millisecond*10)

	if !timer.IntervalPassed(now) {
		t.Fatalf("expected interval passed, got interval not passed yet")
	}

	// Call Tick method to shift the nextMessage time
	timer.Tick(now)

	// After Tick, the interval should not have passed again
	if timer.IntervalPassed(now) {
		t.Fatalf("after tick, expected interval not passed yet, got interval passed")
	}

	// Move forward in time beyond the interval again
	now = now.Add(interval + time.Millisecond*10)

	if !timer.IntervalPassed(now) {
		t.Fatalf("after tick and waiting, expected interval passed, got interval not passed yet")
	}
}
