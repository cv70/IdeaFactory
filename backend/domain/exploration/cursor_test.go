package exploration

import (
	"testing"
	"time"
)

func TestParseOrderedCursorSupportsRFC3339(t *testing.T) {
	ts := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	cursor := buildOrderedCursor(ts, "evt-1")
	gotTs, gotID, ok := parseOrderedCursor(cursor)
	if !ok {
		t.Fatal("expected cursor to parse")
	}
	if gotID != "evt-1" {
		t.Fatalf("expected id evt-1, got %s", gotID)
	}
	if !gotTs.Equal(ts) {
		t.Fatalf("expected ts %v, got %v", ts, gotTs)
	}
}

func TestParseOrderedCursorSupportsUnixMs(t *testing.T) {
	ts := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	cursor := buildOrderedCursorUnixMilli(ts, "evt-2")
	gotTs, gotID, ok := parseOrderedCursor(cursor)
	if !ok {
		t.Fatal("expected cursor to parse")
	}
	if gotID != "evt-2" {
		t.Fatalf("expected id evt-2, got %s", gotID)
	}
	if gotTs.UnixMilli() != ts.UnixMilli() {
		t.Fatalf("expected unix ms %d, got %d", ts.UnixMilli(), gotTs.UnixMilli())
	}
}
