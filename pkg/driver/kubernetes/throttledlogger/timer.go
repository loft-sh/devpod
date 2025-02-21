package throttledlogger

import "time"

type Timer struct {
	nextMessage  time.Time
	tickInterval time.Duration
}

func NewTimer(tickInterval time.Duration) *Timer {
	return &Timer{
		nextMessage:  time.Now().Add(tickInterval),
		tickInterval: tickInterval,
	}
}

func (t *Timer) Tick(now time.Time) {
	t.nextMessage = now.Add(t.tickInterval)
}

func (t *Timer) IntervalPassed(now time.Time) bool {
	return now.After(t.nextMessage)
}
