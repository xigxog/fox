package repo

import (
	"os"
	"path/filepath"

	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/api/kubernetes/v1alpha1"
)

func (r *repo) Publish(deployName string) *v1alpha1.Deployment {
	compsDirPath := filepath.Join(r.cfg.Flags.RepoPath, ComponentsDirName)
	compsDir, err := os.ReadDir(compsDirPath)
	if err != nil {
		log.Fatal("Error listing components dir '%s': %v", compsDirPath, err)
	}

	for _, compDir := range compsDir {
		if !compDir.IsDir() {
			continue
		}
		r.Build(compDir.Name())
	}

	if !r.cfg.Flags.SkipDeploy && deployName != "" {
		return r.Deploy(deployName)
	}
	return nil
}
