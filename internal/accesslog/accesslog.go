// Package accesslog provides HTTP request logging through a bufio.Writer.
//
// This is a genuine case for buffered I/O, unlike cmd/mcp-fetch's stdio
// transport (see mark3labs/mcp-go's StdioServer.writeResponse): each
// request produces one short, independent log line, and under real
// traffic that's many small writes to the same underlying writer -- the
// exact pattern buffering exists to coalesce into fewer syscalls. A
// request-response protocol where each write is already a single,
// necessarily-immediate message (like the MCP case) doesn't have that
// pattern; an access log does.
package accesslog

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Logger buffers access log lines and flushes them periodically, rather
// than issuing a syscall per request. Buffering means a line is not
// guaranteed to reach the underlying writer until the next periodic
// flush or an explicit Flush/Close -- a crash or kill -9 between flushes
// loses at most flushInterval's worth of lines. That trade-off (fewer
// syscalls under load, in exchange for a small window of loggable-but-lost
// lines on an unclean exit) is the actual trade-off buffered I/O makes,
// not a bug.
type Logger struct {
	mu   sync.Mutex
	buf  *bufio.Writer
	stop chan struct{}
	done chan struct{}
}

// New starts a Logger that buffers writes to w and flushes them every
// flushInterval on a background goroutine. Call Close when done to stop
// that goroutine and flush any remaining buffered lines.
func New(w io.Writer, flushInterval time.Duration) *Logger {
	l := &Logger{
		buf:  bufio.NewWriter(w),
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	go l.flushLoop(flushInterval)
	return l
}

func (l *Logger) flushLoop(interval time.Duration) {
	defer close(l.done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.Flush()
		case <-l.stop:
			return
		}
	}
}

// Log appends a single line to the buffer. Safe for concurrent use --
// concurrent HTTP handlers log at the same time, and each call must
// write its whole line atomically so lines from different requests never
// interleave mid-line.
func (l *Logger) Log(line string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(l.buf, line)
}

// Flush writes any buffered lines to the underlying writer immediately.
func (l *Logger) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Flush()
}

// Close stops the background flush goroutine and performs one final
// flush so no buffered lines are lost on a clean shutdown.
func (l *Logger) Close() error {
	close(l.stop)
	<-l.done
	return l.Flush()
}

// statusRecorder wraps http.ResponseWriter to capture the status code
// that was actually written, since http.ResponseWriter has no getter for
// it and the access log needs it after the handler has already run.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Middleware wraps next, logging one line per request: method, path,
// status code, and duration. The status code defaults to 200, matching
// net/http's own behavior when a handler never calls WriteHeader
// explicitly (e.g. the 204 No Content handlers that only call
// w.WriteHeader(http.StatusNoContent) do get captured; a handler that
// writes a 200 body without calling WriteHeader at all also correctly
// logs 200, since that's what net/http sends in that case too).
func (l *Logger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		l.Log(fmt.Sprintf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start)))
	})
}
