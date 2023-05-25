package repo

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	pack "github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/xigxog/kubefox-cli/internal/config"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/api/admin/v1alpha1"
	"github.com/xigxog/kubefox/libs/core/api/common"
	"github.com/xigxog/kubefox/libs/core/api/maker"
	"github.com/xigxog/kubefox/libs/core/validator"
	"sigs.k8s.io/yaml"
)

type repo struct {
	cfg  *config.Config
	path string

	gitRepo *git.Repository
	pack    *pack.Client
}

func New(cfg *config.Config) *repo {
	path := config.Flags.SysRepoPath
	log.Verbose("Opening git repo '%s'", path)

	gitRepo, err := git.PlainOpen(path)
	if err != nil {
		log.Fatal("Error opening system git repo '%s': %v", path, err)
	}

	pack, err := pack.NewClient(pack.WithLogger(log.NewPackLogger()))
	if err != nil {
		log.Fatal("Error creating Buildpack client: %v", err)
	}

	return &repo{
		cfg:     cfg,
		path:    path,
		gitRepo: gitRepo,
		pack:    pack,
	}
}

func (r *repo) CommitAll(msg string) string {
	w, err := r.gitRepo.Worktree()
	if err != nil {
		log.Fatal("Error accessing git worktree: %v", err)
	}
	if _, err = w.Add("."); err != nil {
		log.Fatal("Error adding files to worktree: %v", err)
	}
	hash, err := w.Commit(msg, &git.CommitOptions{})
	if err != nil {
		log.Fatal("Error committing changes: %v", err)
	}
	log.Verbose("Changes committed; hash: %s", hash)

	return hash.String()
}

func (r *repo) BuildComp(comp string) string {
	// replace default keychain to use GitHub token for registry authentication
	kc := &kubefoxKeychain{
		defaultKeychain: authn.DefaultKeychain,
		registry:        config.Flags.Registry,
		authToken:       base64.StdEncoding.EncodeToString([]byte("kubefox:" + r.cfg.GitHub.Token)),
	}
	authn.DefaultKeychain = kc

	path := filepath.Join(r.path, "components", comp)
	if _, err := os.Stat(path); err != nil {
		log.Fatal("Error opening component dir '%s': %v", path, err)
	}

	now := time.Now()
	img := r.GetContainerImage(comp)
	buildOpts := pack.BuildOptions{
		Interactive:  false,
		CreationTime: &now,
		AppPath:      path,
		Image:        img,
		Builder:      config.Flags.Builder,
		Publish:      config.Flags.PublishImage,
		ClearCache:   config.Flags.ClearCache,
		PullPolicy:   image.PullIfNotPresent,
	}

	log.Info("Building image '%s' for component '%s'", img, comp)
	if err := r.pack.Build(context.Background(), buildOpts); err != nil {
		log.Fatal("Error building component: %v", err)
	}

	return img
}

func (r *repo) GenerateSysObj() *v1alpha1.System {
	org := r.cfg.GitHub.Org.Name
	sys := config.Flags.System

	appDirPath := filepath.Join(config.Flags.SysRepoPath, "apps")
	appDir, err := os.ReadDir(appDirPath)
	if err != nil {
		log.Fatal("Error listing app dirs '%s': %v", appDirPath, err)
	}

	sysObj := maker.New[v1alpha1.System](maker.Props{Name: config.Flags.System})
	sysObj.GitRepo = fmt.Sprintf("https://github.com/%s/%s.git", org, sys)
	sysObj.GitRef = r.GetRefName()
	sysObj.GitHash = r.GetHash("")
	sysObj.Message = config.Flags.Msg

	apps := map[string]*common.App{}
	for _, appDir := range appDir {
		if !appDir.IsDir() {
			continue
		}

		appName := appDir.Name()
		appYamlPath := filepath.Join(appDirPath, appName, "app.yaml")
		appYaml, err := os.ReadFile(appYamlPath)
		if err != nil {
			log.Fatal("Error reading app yaml '%s': %v", appYamlPath, err)
		}
		app := &common.App{}
		err = yaml.Unmarshal(appYaml, app)
		if err != nil {
			log.Fatal("Error parsing app yaml '%s': %v", appYamlPath, err)
		}
		app.GitHash = r.GetHash("apps/" + appName)

		for compName, comp := range app.Components {
			comp.GitHash = r.GetCompHash(compName)
			comp.Image = r.GetContainerImage(compName)
		}

		apps[appName] = app
	}
	sysObj.Apps = apps

	sysObj.Id = "na"
	v := validator.New(log.Logger())
	if errs := v.Validate(sysObj); errs != nil {
		log.Marshal(errs)
		log.Fatal("System object is not valid")
	}
	sysObj.Id = ""

	return sysObj
}

func (r *repo) GetContainerImage(comp string) string {
	org := r.cfg.GitHub.Org.Name
	reg := config.Flags.Registry
	sys := config.Flags.System
	hash := r.GetCompHash(comp)

	return fmt.Sprintf("%s/%s/%s/%s:%s", reg, org, sys, comp, hash)
}

func (r *repo) GetRefName() string {
	gitRef, err := r.gitRepo.Head()
	if err != nil {
		log.Fatal("Error opening head ref of git repo: %v", err)
	}

	if gitRef.Name().IsBranch() {
		return "branch/" + gitRef.Name().Short()
	}

	// find tag
	var refName string
	tags, err := r.gitRepo.Tags()
	if err != nil {
		return ""
	}
	tags.ForEach(func(tag *plumbing.Reference) error {
		if gitRef.Hash() == tag.Hash() {
			refName = "tag/" + tag.Name().Short()
		}
		return nil
	})

	return refName
}

func (r *repo) GetCompHash(comp string) string {
	return r.GetHash(filepath.Join("components", comp))
}

func (r *repo) GetHash(subPath string) string {
	if !r.IsClean() {
		log.Fatal("Error finding hash: uncommitted changes present")
	}

	path := filepath.Join(r.path, subPath)
	iter, err := r.gitRepo.Log(&git.LogOptions{
		PathFilter: func(c string) bool {
			return strings.HasPrefix(c, subPath)
		},
	})
	if err != nil {
		log.Fatal("Error finding hash for path '%s': %v", path, err)
	}

	commit, err := iter.Next()
	if err != nil {
		log.Fatal("Error finding hash for path '%s': %v", path, err)
	}
	if commit == nil {
		log.Fatal("Error finding hash for path '%s': no commits have been made", path)
	}

	return commit.Hash.String()[0:7]
}

func (r *repo) IsClean() bool {
	w, err := r.gitRepo.Worktree()
	if err != nil {
		log.Fatal("Error accessing git worktree: %v", err)
	}
	s, err := w.Status()
	if err != nil {
		log.Fatal("Error getting git worktree status: %v", err)
	}

	return s.IsClean()
}
