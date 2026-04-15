package rag

import (
	"fmt"
	"os"
	"path/filepath"
)

// LoadDir loads *.md from a host directory (flat) into an index.
func LoadDir(root string) (*Index, error) {
	root = filepath.Clean(root)
	fi, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("knowledge dir: %w", err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("knowledge path is not a directory: %s", root)
	}
	return LoadFS(os.DirFS(root), ".")
}
