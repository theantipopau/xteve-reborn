package src

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestWebLogKeepsNewestLines guards the trim bug where, once the log was
// full, the loop bound (i < limit instead of i < len) threw away the newest
// line on every other append - so recent errors could silently vanish.
func TestWebLogKeepsNewestLines(t *testing.T) {

	var oldLimit = Settings.LogEntriesRAM
	defer func() { Settings.LogEntriesRAM = oldLimit }()

	Settings.LogEntriesRAM = 10
	resetWebLog()

	for i := 0; i < 25; i++ {
		appendWebLog(fmt.Sprintf("line %d", i))
	}

	var snapshot = snapshotWebLog()

	if len(snapshot.Log) != 10 {
		t.Fatalf("got %d lines, want 10", len(snapshot.Log))
	}

	for i, line := range snapshot.Log {
		var want = fmt.Sprintf("line %d", 15+i)
		if !strings.HasSuffix(line, want) {
			t.Errorf("line %d = %q, want suffix %q", i, line, want)
		}
	}
}

// TestWebLogCountsErrorsAndWarnings checks the counters shown on the
// dashboard stay in sync with what's actually in the (trimmed) log.
func TestWebLogCountsErrorsAndWarnings(t *testing.T) {

	var oldLimit = Settings.LogEntriesRAM
	defer func() { Settings.LogEntriesRAM = oldLimit }()

	Settings.LogEntriesRAM = 3
	resetWebLog()

	appendWebLog("[x] [ERROR] old")
	appendWebLog("[x] [WARNING] one")
	appendWebLog("[x] [WARNING] two")
	appendWebLog("[x] info")

	var snapshot = snapshotWebLog()

	if snapshot.Errors != 0 || snapshot.Warnings != 2 {
		t.Errorf("errors=%d warnings=%d, want 0 and 2 (the ERROR line was trimmed)", snapshot.Errors, snapshot.Warnings)
	}
}

// TestWebLogHasNoHTMLEntities: the Log page renders lines with textContent
// (so provider-controlled text can't inject markup), which means any
// &nbsp; encoding done on the server shows up literally on screen.
func TestWebLogHasNoHTMLEntities(t *testing.T) {

	resetWebLog()
	showInfo("Streaming URL:http://example.com/stream")

	for _, line := range snapshotWebLog().Log {
		if strings.Contains(line, "&nbsp;") {
			t.Errorf("log line contains HTML entity: %q", line)
		}
	}
}

// TestWebLogConcurrentAccess mirrors real traffic: many stream handlers
// logging at once while WS responses snapshot the log.
func TestWebLogConcurrentAccess(t *testing.T) {

	var oldLimit = Settings.LogEntriesRAM
	defer func() { Settings.LogEntriesRAM = oldLimit }()

	Settings.LogEntriesRAM = 50
	resetWebLog()

	var wg sync.WaitGroup

	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				appendWebLog(fmt.Sprintf("writer %d line %d", n, i))
			}
		}(w)
	}

	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				_ = snapshotWebLog()
			}
		}()
	}

	wg.Wait()

	if got := len(snapshotWebLog().Log); got != 50 {
		t.Errorf("got %d lines, want 50", got)
	}
}
