package peer

import "time"

// ConnStats encapsulates per-connection counters and timing aggregation.
// It provides helper methods to emit summaries via ConnObserver.
// All methods are not concurrency-safe and expected to be used within the loop goroutine.

type ConnStats struct {
	readCount     int64
	writeCount    int64
	writeErrCount int64
	writeSum      time.Duration
	writeMax      time.Duration
}

func (s *ConnStats) OnDelivered() {
	s.readCount++
}

func (s *ConnStats) OnWriteResult(dur time.Duration, err error) {
	if err != nil {
		s.writeErrCount++
		return
	}
	s.writeCount++
	s.writeSum += dur
	if dur > s.writeMax {
		s.writeMax = dur
	}
}

func (s *ConnStats) HasActivity() bool {
	return s.readCount > 0 || s.writeCount > 0 || s.writeErrCount > 0
}

func (s *ConnStats) Avg() time.Duration {
	if s.writeCount == 0 {
		return 0
	}
	return time.Duration(int64(s.writeSum) / s.writeCount)
}

func (s *ConnStats) Summary(obs ConnObserver, url string, reconnections int64) {
	if !s.HasActivity() {
		return
	}
	obs.Summary(url, s.readCount, s.writeCount, s.writeErrCount, s.Avg(), s.writeMax, reconnections)
}

func (s *ConnStats) Exit(obs ConnObserver, url, reason string, err error, reconnections int64) {
	if !s.HasActivity() {
		return
	}
	obs.Exit(url, reason, err, s.readCount, s.writeCount, s.writeErrCount, s.Avg(), s.writeMax, reconnections)
}

// Reconnections computes the number of reconnection events from a total connection count.
// It returns max(total-1, 0).
func Reconnections(total int64) int64 {
	r := total - 1
	if r < 0 {
		r = 0
	}
	return r
}
