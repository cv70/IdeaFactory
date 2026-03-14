package exploration

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseOrderedCursor parses "<time>|<id>" where time is either RFC3339 or unix milliseconds.
func parseOrderedCursor(cursor string) (time.Time, string, bool) {
	parts := strings.SplitN(cursor, "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", false
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	if left == "" || right == "" {
		return time.Time{}, "", false
	}
	if ts, err := time.Parse(time.RFC3339, left); err == nil {
		return ts, right, true
	}
	if ms, err := strconv.ParseInt(left, 10, 64); err == nil {
		return time.UnixMilli(ms), right, true
	}
	return time.Time{}, "", false
}

func buildOrderedCursor(ts time.Time, id string) string {
	return ts.UTC().Format(time.RFC3339) + "|" + id
}

func buildOrderedCursorUnixMilli(ts time.Time, id string) string {
	return fmt.Sprintf("%d|%s", ts.UnixMilli(), id)
}
