package common

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type StreamEndReason string

const (
	StreamEndReasonNone       StreamEndReason = ""
	StreamEndReasonDone       StreamEndReason = "done"
	StreamEndReasonTimeout    StreamEndReason = "timeout"
	StreamEndReasonClientGone StreamEndReason = "client_gone"
	// StreamEndReasonDrained：客户端已断开，但 DrainUpstreamOnClientDisconnect
	// 仍把上游读到了正常终态（[DONE]/EOF）。上游到底是 [DONE] 还是 EOF 记在
	// DrainedFrom，需要区分的地方（首字后错误准则 4）用 UpstreamEndReason()。
	StreamEndReasonDrained         StreamEndReason = "drained"
	StreamEndReasonScannerErr      StreamEndReason = "scanner_error"
	StreamEndReasonHandlerStop     StreamEndReason = "handler_stop"
	StreamEndReasonEOF             StreamEndReason = "eof"
	StreamEndReasonPanic           StreamEndReason = "panic"
	StreamEndReasonPingFail        StreamEndReason = "ping_fail"
	StreamEndReasonFirstTokenError StreamEndReason = "first_token_error"
)

const maxStreamErrorEntries = 20

type StreamErrorEntry struct {
	Message   string
	Timestamp time.Time
}

type StreamStatus struct {
	EndReason StreamEndReason
	EndError  error
	endOnce   sync.Once

	// ClientGone records that the client disconnected while the upstream was
	// still streaming, whether or not the upstream was then drained.
	ClientGone bool
	// DrainedFrom is the upstream terminal (done or eof) that a successful
	// client-disconnect drain was relabelled from.
	DrainedFrom StreamEndReason
	// DrainDuration is how long the upstream was read after the client left.
	DrainDuration time.Duration

	mu         sync.Mutex
	Errors     []StreamErrorEntry
	ErrorCount int
}

func NewStreamStatus() *StreamStatus {
	return &StreamStatus{}
}

func (s *StreamStatus) SetEndReason(reason StreamEndReason, err error) {
	if s == nil {
		return
	}
	s.endOnce.Do(func() {
		s.EndReason = reason
		s.EndError = err
	})
}

// OverrideEndReason replaces the first-wins end reason after every stream
// goroutine has exited. Used for Responses first-token-error billing and for
// remapping a successful client-disconnect drain to StreamEndReasonDrained.
func (s *StreamStatus) OverrideEndReason(reason StreamEndReason, err error) {
	if s == nil {
		return
	}
	s.EndReason = reason
	s.EndError = err
}

// MarkDrained relabels a stream whose upstream reached its own end (done or
// eof) after the client disconnected. Any other end reason, such as a handler
// stop or a scanner error, is kept because the drain did not complete.
// Call it only after every stream goroutine has exited.
func (s *StreamStatus) MarkDrained(drainDuration time.Duration) bool {
	if s == nil {
		return false
	}
	s.ClientGone = true
	s.DrainDuration = drainDuration
	if s.EndReason != StreamEndReasonDone && s.EndReason != StreamEndReasonEOF {
		return false
	}
	s.DrainedFrom = s.EndReason
	s.OverrideEndReason(StreamEndReasonDrained, nil)
	return true
}

// UpstreamEndReason is how the upstream side of the stream ended: the done or
// eof a drained stream reached, otherwise EndReason.
func (s *StreamStatus) UpstreamEndReason() StreamEndReason {
	if s == nil {
		return StreamEndReasonNone
	}
	if s.EndReason == StreamEndReasonDrained && s.DrainedFrom != StreamEndReasonNone {
		return s.DrainedFrom
	}
	return s.EndReason
}

func (s *StreamStatus) RecordError(msg string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorCount++
	if len(s.Errors) < maxStreamErrorEntries {
		s.Errors = append(s.Errors, StreamErrorEntry{
			Message:   msg,
			Timestamp: time.Now(),
		})
	}
}

func (s *StreamStatus) HasErrors() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ErrorCount > 0
}

func (s *StreamStatus) TotalErrorCount() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ErrorCount
}

func (s *StreamStatus) IsNormalEnd() bool {
	if s == nil {
		return true
	}
	return s.EndReason == StreamEndReasonDone ||
		s.EndReason == StreamEndReasonEOF ||
		s.EndReason == StreamEndReasonHandlerStop ||
		s.EndReason == StreamEndReasonDrained
}

func (s *StreamStatus) Summary() string {
	if s == nil {
		return "StreamStatus<nil>"
	}
	b := &strings.Builder{}
	fmt.Fprintf(b, "reason=%s", s.EndReason)
	if s.DrainedFrom != StreamEndReasonNone {
		fmt.Fprintf(b, " upstream_end=%s", s.DrainedFrom)
	}
	if s.DrainDuration > 0 {
		fmt.Fprintf(b, " drain=%s", s.DrainDuration.Round(time.Millisecond))
	}
	if s.EndError != nil {
		fmt.Fprintf(b, " end_error=%q", s.EndError.Error())
	}
	s.mu.Lock()
	if s.ErrorCount > 0 {
		fmt.Fprintf(b, " soft_errors=%d", s.ErrorCount)
	}
	s.mu.Unlock()
	return b.String()
}
