package repo

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	pack "github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/xigxog/kubefox-cli/internal/config"
	"github.com/xigxog/kubefox-cli/internal/log"
)

func (r *repo) BuildComp(comp string) string {
	// replace default keychain to use GitHub token for registry authentication
	kc := &kubefoxKeychain{
		defaultKeychain: authn.DefaultKeychain,
		registry:        r.cfg.ContainerRegistry.Address,
	}
	if r.cfg.GitHub.Token != "" {
		kc.authToken = base64.StdEncoding.EncodeToString([]byte("kubefox:" + r.cfg.GitHub.Token))
	} else {
		kc.authToken = r.cfg.ContainerRegistry.Token
	}
	authn.DefaultKeychain = kc

	path := filepath.Join(r.path, "components", comp)
	if _, err := os.Stat(path); err != nil {
		log.Fatal("Error opening component dir '%s': %v", path, err)
	}

	localReg := strings.HasPrefix(r.cfg.ContainerRegistry.Address, config.LocalRegistry)
	if localReg {
		log.Verbose("Local registry is set, publish will be skipped.")
	}

	now := time.Now()
	img := r.GetContainerImage(comp)
	publish := config.Flags.PublishImage && !localReg
	buildOpts := pack.BuildOptions{
		Interactive:  false,
		CreationTime: &now,
		AppPath:      path,
		Image:        img,
		Builder:      config.Flags.Builder,
		Publish:      publish,
		ClearCache:   config.Flags.ClearCache,
		PullPolicy:   image.PullIfNotPresent,
	}

	log.Info("Building image '%s' for component '%s'", img, comp)
	if err := r.pack.Build(context.Background(), buildOpts); err != nil {
		log.Fatal("Error building component: %v", err)
	}

	kind := config.Flags.Kind
	if kind == "" && r.cfg.Kind.AlwaysLoad {
		kind = r.cfg.Kind.ClusterName
	}
	if kind != "" {
		log.Info("Loading component image '%s' into Kind cluster '%s'", img, kind)
		cmd := exec.Command("kind", "load", "docker-image", "--name="+kind, img)
		if err := cmd.Run(); err != nil {
			log.Error("Error loading component image into Kind: %v", err)
		}
	}

	return img
}
