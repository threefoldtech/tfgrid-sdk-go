package peer

import (
	"time"
)

// ConnObserver abstracts logging/observability from the transport logic.
// Implementations should be lightweight and non-blocking. All methods are best-effort.
// A no-op implementation is provided by default.

type ConnObserver interface {
	// ReaderBackpressure is called when forwarding from the websocket reader to the internal reader channel blocks.
	ReaderBackpressure(url string, readerLen, readerCap int)

	// OutputBackpressure is called when forwarding from the internal outputCh to the peer-wide output blocks.
	OutputBackpressure(url string, outputLen, outputCap, outputChLen, outputChCap, writerLen, writerCap int)

	// MaybeWriteSpike receives each write result and may emit a spike warning based on internal thresholds.
	// Implementations decide whether to log (e.g., duration > 100ms or writer queue >75%).
	MaybeWriteSpike(url string, writeDur time.Duration, writerLen, writerCap, outputLen, outputCap, outputChLen, outputChCap int)

	// Summary logs the per-connection periodic summary.
	Summary(url string, reads, writes, writeErrors int64, avg, max time.Duration, reconnections int64)

	// Exit logs a per-connection exit summary.
	Exit(url, reason string, err error, reads, writes, writeErrors int64, avg, max time.Duration, reconnections int64)
}

// A no-op implementation
type noopObserver struct{}

func (noopObserver) ReaderBackpressure(string, int, int)                                      {}
func (noopObserver) OutputBackpressure(string, int, int, int, int, int, int)                  {}
func (noopObserver) MaybeWriteSpike(string, time.Duration, int, int, int, int, int, int)      {}
func (noopObserver) Summary(string, int64, int64, int64, time.Duration, time.Duration, int64) {}
func (noopObserver) Exit(string, string, error, int64, int64, int64, time.Duration, time.Duration, int64) {
}

// Ensure noopObserver satisfies ConnObserver to avoid unused warnings
// TODO: Currently NewLogObserver() is what NewConnection() currently sets by default; We meed to support swap implementations.
var _ ConnObserver = (*noopObserver)(nil)
