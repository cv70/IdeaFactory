package exploration

import (
	"fmt"
	"strconv"
	"strings"
)

func formatWorkspaceID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

func parseWorkspaceID(workspaceID string) (uint, error) {
	raw := strings.TrimSpace(workspaceID)
	if raw == "" {
		return 0, fmt.Errorf("workspace id is required")
	}
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("workspace id must be a positive integer")
	}
	return uint(parsed), nil
}
