package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

// LoadDirFromPath loads *.yaml skill packs from a host directory.
func LoadDirFromPath(root string) (*Registry, error) {
	root = filepath.Clean(root)
	fi, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("skills dir: %w", err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("skills path is not a directory: %s", root)
	}
	return LoadDir(os.DirFS(root), ".")
}
