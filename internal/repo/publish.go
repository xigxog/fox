package repo

import (
	"os"
	"path/filepath"

	"github.com/xigxog/kubefox-cli/internal/log"
)

func (r *repo) Publish() {
	compsDirPath := filepath.Join(r.cfg.Flags.RepoPath, ComponentsDirName)
	compsDir, err := os.ReadDir(compsDirPath)
	if err != nil {
		log.Fatal("Error listing components dir '%s': %v", compsDirPath, err)
	}

	for _, compDir := range compsDir {
		if !compDir.IsDir() {
			continue
		}

		r.BuildComp(compDir.Name())
	}
}
