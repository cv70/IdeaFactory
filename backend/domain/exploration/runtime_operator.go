package exploration

import (
	"context"
	"os"
	"path/filepath"
)

type RuntimeOperator interface {
	PrepareWorkDir(ctx context.Context, workDir string) error
}

type LocalRuntimeOperator struct{}

func (LocalRuntimeOperator) PrepareWorkDir(_ context.Context, workDir string) error {
	return os.MkdirAll(filepath.Clean(workDir), 0o755)
}
