package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pack "github.com/buildpacks/pack/pkg/client"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/xigxog/kubefox-cli/internal/config"
	"github.com/xigxog/kubefox-cli/internal/kubernetes"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/utils"
	"gopkg.in/yaml.v2"
)

type repo struct {
	cfg  *config.Config
	app  *App
	path string

	gitRepo *git.Repository
	k8s     *kubernetes.Client
	pack    *pack.Client
}

type App struct {
	Title             string `json:"title,omitempty"`
	Description       string `json:"description,omitempty"`
	Name              string `json:"name"`
	ContainerRegistry string `json:"containerRegistry,omitempty"`
}

func ReadApp(repoPath string) (*App, error) {
	log.Verbose("Reading app definition '%s/app.yaml'", repoPath)
	b, err := os.ReadFile(filepath.Join(repoPath, "app.yaml"))
	if err != nil {
		return nil, err
	}
	app := &App{}
	if err := yaml.Unmarshal(b, app); err != nil {
		return nil, err
	}
	if app.Name == "" || app.Name != utils.Clean(app.Name) {
		return nil, fmt.Errorf("invalid app name")
	}
	return app, nil
}

func New(cfg *config.Config) *repo {
	path := config.Flags.RepoPath

	app, err := ReadApp(path)
	if err != nil {
		log.Fatal("Error reading the Repo's 'app.yaml', try running 'fox init': %v", err)
	}

	log.Verbose("Opening git repo '%s'", path)
	gitRepo, err := git.PlainOpen(path)
	if err != nil {
		log.Fatal("Error opening git repo '%s': %v", path, err)
	}

	pack, err := pack.NewClient(pack.WithLogger(log.NewPackLogger()))
	if err != nil {
		log.Fatal("Error creating Buildpack client: %v", err)
	}

	return &repo{
		cfg:     cfg,
		app:     app,
		path:    path,
		gitRepo: gitRepo,
		k8s:     kubernetes.NewClient(),
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
	log.Verbose("Changes committed; commit hash '%s'", hash)

	return hash.String()
}

func (r *repo) GetContainerImage(comp string) string {
	return fmt.Sprintf("%s/%s/%s:%s", r.cfg.ContainerRegistry.Address, r.app.Name, comp, r.GetCompCommit(comp))
}

func (r *repo) GetRepoURL() string {
	o, err := r.gitRepo.Remote("origin")
	if err != nil || len(o.Config().URLs) == 0 {
		return ""
	}

	return o.Config().URLs[0]
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

func (r *repo) GetCompCommit(comp string) string {
	return r.GetCommit(filepath.Join("components", comp))
}

func (r *repo) GetCommit(subPath string) string {
	if !r.IsClean() {
		log.Fatal("Error finding commit hash: uncommitted changes present")
	}

	path := filepath.Join(r.path, subPath)
	iter, err := r.gitRepo.Log(&git.LogOptions{
		PathFilter: func(c string) bool {
			return strings.HasPrefix(c, subPath)
		},
	})
	if err != nil {
		log.Fatal("Error finding commit hash for path '%s': %v", path, err)
	}

	commit, err := iter.Next()
	if err != nil {
		log.Fatal("Error finding commit hash for path '%s': %v", path, err)
	}
	if commit == nil {
		log.Fatal("Error finding commit hash for path '%s': no commits have been made", path)
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
