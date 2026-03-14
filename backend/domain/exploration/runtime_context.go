package exploration

import (
	"context"
	"errors"
	"path/filepath"
)

type runtimeContextKey struct{}

type RuntimeContextData struct {
	WorkspaceID string
	RunID       string
	PlanID      string
	WorkDir     string
	InputDigest string
}

func InitRuntimeContext(ctx context.Context, data RuntimeContextData) context.Context {
	return context.WithValue(ctx, runtimeContextKey{}, data)
}

func RuntimeContextFrom(ctx context.Context) (RuntimeContextData, bool) {
	v := ctx.Value(runtimeContextKey{})
	if v == nil {
		return RuntimeContextData{}, false
	}
	data, ok := v.(RuntimeContextData)
	return data, ok
}

func ResolveWorkFile(ctx context.Context, relative string) (string, error) {
	data, ok := RuntimeContextFrom(ctx)
	if !ok || data.WorkDir == "" {
		return "", errors.New("runtime workdir not found in context")
	}
	return filepath.Clean(filepath.Join(data.WorkDir, relative)), nil
}
