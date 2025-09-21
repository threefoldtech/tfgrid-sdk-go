package peer

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	// WarnSamplePeriod controls burst sampling window for warn-level logs.
	WarnSamplePeriod = 5 * time.Second
	// WriteSpikeWarnThreshold defines the write duration threshold to warn for spikes.
	WriteSpikeWarnThreshold = 100 * time.Millisecond
	// WriterQueueWarnPercent defines when to warn on writer queue usage percentage.
	WriterQueueWarnPercent = 75
)

// NewLogObserver builds a zerolog-based observer with the approved sampling.
func NewLogObserver(url string) ConnObserver {
	// reader/output backpressure: 5s after first
	readerSampler := zerolog.LevelSampler{WarnSampler: &zerolog.BurstSampler{Burst: 1, Period: WarnSamplePeriod}}
	bpReaderLog := log.With().Str("url", url).Logger().Sample(&readerSampler)

	outputSampler := zerolog.LevelSampler{WarnSampler: &zerolog.BurstSampler{Burst: 1, Period: WarnSamplePeriod}}
	bpOutLog := log.With().Str("url", url).Logger().Sample(&outputSampler)

	// write spike: 5s after first
	spikeSampler := zerolog.LevelSampler{WarnSampler: &zerolog.BurstSampler{Burst: 1, Period: WarnSamplePeriod}}
	spikeLog := log.With().Str("url", url).Logger().Sample(&spikeSampler)

	return &logObserver{
		url:         url,
		bpReaderLog: bpReaderLog,
		bpOutLog:    bpOutLog,
		spikeLog:    spikeLog,
	}
}

type logObserver struct {
	url         string
	bpReaderLog zerolog.Logger
	bpOutLog    zerolog.Logger
	spikeLog    zerolog.Logger
}

func (o *logObserver) ReaderBackpressure(_ string, readerLen, readerCap int) {
	o.bpReaderLog.Warn().
		Int("reader_len", readerLen).Int("reader_cap", readerCap).
		Msg("reader channel backpressure (blocking)")
}

func (o *logObserver) OutputBackpressure(_ string, outputLen, outputCap, outputChLen, outputChCap, writerLen, writerCap int) {
	o.bpOutLog.Warn().
		Int("output_len", outputLen).Int("output_cap", outputCap).
		Int("outputCh_len", outputChLen).Int("outputCh_cap", outputChCap).
		Int("writer_len", writerLen).Int("writer_cap", writerCap).
		Msg("output channel backpressure (blocking)")
}

func (o *logObserver) MaybeWriteSpike(_ string, writeDur time.Duration, writerLen, writerCap, outputLen, outputCap, outputChLen, outputChCap int) {
	// Thresholds preserved from previous behavior: duration > 100ms or writer queue > 75% full.
	if writeDur > WriteSpikeWarnThreshold || (writerCap > 0 && writerLen*100 > writerCap*WriterQueueWarnPercent) {
		o.spikeLog.Warn().
			Dur("write_dur", writeDur).
			Int("writer_len", writerLen).Int("writer_cap", writerCap).
			Int("output_len", outputLen).Int("output_cap", outputCap).
			Int("outputCh_len", outputChLen).Int("outputCh_cap", outputChCap).
			Msg("write spike / queue saturation")
	}
}

func (o *logObserver) Summary(url string, reads, writes, writeErrors int64, avg, max time.Duration, reconnections int64) {
	log.Debug().
		Str("url", url).
		Int64("reads", reads).
		Int64("writes", writes).
		Int64("write_errors", writeErrors).
		Dur("avg", avg).
		Dur("max", max).
		Int64("reconnections", reconnections).
		Msg("relay loop summary")
}

func (o *logObserver) Exit(url, reason string, err error, reads, writes, writeErrors int64, avg, max time.Duration, reconnections int64) {
	log.Debug().
		Str("url", url).
		Str("reason", reason).
		Err(err).
		Int64("reads", reads).
		Int64("writes", writes).
		Int64("write_errors", writeErrors).
		Dur("avg", avg).
		Dur("max", max).
		Int64("reconnections", reconnections).
		Msg("relay loop exited")
}
