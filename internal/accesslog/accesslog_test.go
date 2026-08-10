package accesslog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLogBuffersUntilFlush(t *testing.T) {
	var underlying bytes.Buffer
	// A long flush interval means the background goroutine won't flush
	// during this test -- any content in `underlying` before we call
	// Flush ourselves proves buffering isn't happening.
	l := New(&underlying, time.Hour)
	defer l.Close()

	l.Log("line one")
	l.Log("line two")

	if underlying.Len() != 0 {
		t.Fatalf("expected nothing written to the underlying writer before Flush, got %q", underlying.String())
	}

	if err := l.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	got := underlying.String()
	if !strings.Contains(got, "line one") || !strings.Contains(got, "line two") {
		t.Fatalf("expected both lines after Flush, got %q", got)
	}
	if strings.Index(got, "line one") > strings.Index(got, "line two") {
		t.Fatalf("expected line one before line two, got %q", got)
	}
}

func TestPeriodicFlush(t *testing.T) {
	var underlying bytes.Buffer
	l := New(&underlying, 10*time.Millisecond)
	defer l.Close()

	l.Log("periodic line")

	deadline := time.After(2 * time.Second)
	for {
		l.mu.Lock()
		content := underlying.String()
		l.mu.Unlock()
		if strings.Contains(content, "periodic line") {
			return
		}
		select {
		case <-deadline:
			t.Fatal("line was not flushed by the periodic flush loop within 2s")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestCloseFlushesRemainingLines(t *testing.T) {
	var underlying bytes.Buffer
	l := New(&underlying, time.Hour)

	l.Log("final line")
	if underlying.Len() != 0 {
		t.Fatalf("expected nothing written before Close, got %q", underlying.String())
	}

	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !strings.Contains(underlying.String(), "final line") {
		t.Fatalf("expected Close to flush remaining lines, got %q", underlying.String())
	}
}

func TestConcurrentLogDoesNotInterleaveOrDropLines(t *testing.T) {
	var underlying bytes.Buffer
	l := New(&underlying, time.Hour)

	const goroutines = 20
	const linesEach = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < linesEach; i++ {
				l.Log(strings.Repeat("x", 40)) // fixed-width line so a corrupted interleave would change line count
			}
		}(g)
	}
	wg.Wait()

	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	lines := strings.Split(strings.TrimRight(underlying.String(), "\n"), "\n")
	if len(lines) != goroutines*linesEach {
		t.Fatalf("expected %d lines, got %d (interleaving or drops under concurrency)", goroutines*linesEach, len(lines))
	}
	for _, line := range lines {
		if len(line) != 40 {
			t.Fatalf("expected every line to be exactly 40 chars (no interleaving), got %d: %q", len(line), line)
		}
	}
}

func TestMiddlewareLogsMethodPathAndStatus(t *testing.T) {
	var underlying bytes.Buffer
	l := New(&underlying, time.Hour)

	handler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got := underlying.String()
	if !strings.Contains(got, "POST") || !strings.Contains(got, "/items") || !strings.Contains(got, "201") {
		t.Fatalf("expected log line with method, path, and status, got %q", got)
	}
}

func TestMiddlewareDefaultsToStatus200WhenWriteHeaderNeverCalled(t *testing.T) {
	var underlying bytes.Buffer
	l := New(&underlying, time.Hour)

	handler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok")) // no explicit WriteHeader -- net/http sends 200
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if !strings.Contains(underlying.String(), "200") {
		t.Fatalf("expected default status 200 in log line, got %q", underlying.String())
	}
}
