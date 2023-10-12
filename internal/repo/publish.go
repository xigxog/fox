package repo

import (
	"os"
	"path/filepath"

	"github.com/xigxog/kubefox-cli/internal/config"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/utils"
)

func (r *repo) Publish() {
	config.Flags.PublishImage = true

	compsDirPath := filepath.Join(config.Flags.RepoPath, "components")
	compsDir, err := os.ReadDir(compsDirPath)
	if err != nil {
		log.Fatal("Error listing components dir '%s': %v", compsDirPath, err)
	}

	for _, compDir := range compsDir {
		if !compDir.IsDir() {
			continue
		}

		compName := utils.Clean(compDir.Name())
		r.BuildComp(compName)
	}
}
