package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	docker "github.com/docker/docker/client"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/xigxog/fox/internal/config"
	"github.com/xigxog/fox/internal/kubernetes"
	"github.com/xigxog/fox/internal/log"
	foxutils "github.com/xigxog/fox/internal/utils"
	"github.com/xigxog/kubefox/utils"
	"gopkg.in/yaml.v2"
)

type repo struct {
	cfg *config.Config
	app *App

	rootPath string
	appPath  string

	gitRepo *git.Repository
	k8s     *kubernetes.Client
	docker  *docker.Client
}

type App struct {
	Title             string `json:"title,omitempty"`
	Description       string `json:"description,omitempty"`
	Name              string `json:"name"`
	ContainerRegistry string `json:"containerRegistry,omitempty"`
}

func New(cfg *config.Config) *repo {
	cfg.CleanPaths(false)

	repoPath := cfg.Flags.RepoPath
	appPath := cfg.Flags.AppPath

	if !strings.HasPrefix(appPath, repoPath) {
		log.Fatal("The app is not part of the Git repo.")
	}

	app, err := ReadApp(appPath)
	if err != nil {
		log.Fatal("Error reading the repo's 'app.yaml', try running 'fox init': %v", err)
	}

	log.Verbose("Opening git repo '%s'", repoPath)
	gitRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		log.Fatal("Error opening git repo '%s': %v", repoPath, err)
	}

	d, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal("Error creating Docker client: %v", err)
	}

	return &repo{
		cfg:      cfg,
		app:      app,
		rootPath: repoPath,
		appPath:  appPath,
		gitRepo:  gitRepo,
		k8s:      kubernetes.NewClient(cfg),
		docker:   d,
	}
}

func ReadApp(path string) (*App, error) {
	log.Verbose("Reading app definition '%s'", filepath.Join(path, "app.yaml"))
	b, err := os.ReadFile(filepath.Join(path, "app.yaml"))
	if err != nil {
		return nil, err
	}
	app := &App{}
	if err := yaml.Unmarshal(b, app); err != nil {
		return nil, err
	}
	if app.Name == "" || !utils.IsValidName(app.Name) {
		return nil, fmt.Errorf("invalid app name")
	}
	return app, nil
}

func WriteApp(path string, app *App) {
	appPath := filepath.Join(path, "app.yaml")
	b, err := yaml.Marshal(app)
	if err != nil {
		log.Fatal("Error marshaling app definition: %v", err)
	}
	foxutils.EnsureDirForFile(appPath)
	if err := os.WriteFile(appPath, b, 0644); err != nil {
		log.Fatal("Error writing app definition file: %v", err)
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

func (r *repo) CreateTag(tag string) *plumbing.Reference {
	log.Info("Creating tag '%s'.", tag)
	h, err := r.gitRepo.Head()
	if err != nil {
		log.Fatal("Error opening head ref of git repo: %v", err)
	}
	ref, err := r.gitRepo.CreateTag(tag, h.Hash(), nil)
	if err != nil {
		log.Fatal("Error creating tag '%s': %v", tag, err)
	}
	return ref
}

func (r *repo) GetCompImageFromDir(compDirName string) string {
	name := utils.CleanName(compDirName)
	commit := r.GetCompCommit(compDirName)
	return r.GetCompImage(name, commit.Hash.String())
}

func (r *repo) GetCompImage(name, commit string) string {
	return fmt.Sprintf("%s/%s/%s:%s", r.cfg.ContainerRegistry.Address, r.app.Name, name, commit)
}

func (r *repo) GetRepoURL() string {
	o, err := r.gitRepo.Remote("origin")
	if err != nil || len(o.Config().URLs) == 0 {
		return ""
	}

	return o.Config().URLs[0]
}

func (r *repo) GetHeadRef() string {
	gitRef, err := r.gitRepo.Head()
	if err != nil {
		log.Fatal("Error opening head ref of git repo: %v", err)
	}
	if gitRef.Name().IsBranch() {
		return gitRef.Name().String()
	}

	return ""
}

func (r *repo) GetTagRef() string {
	gitRef, err := r.gitRepo.Head()
	if err != nil {
		log.Fatal("Error opening head ref of git repo: %v", err)
	}
	// find tag
	var refName string
	tags, err := r.gitRepo.Tags()
	if err != nil {
		return ""
	}
	tags.ForEach(func(tag *plumbing.Reference) error {
		if gitRef.Hash() == tag.Hash() {
			refName = tag.Name().String()
		}
		return nil
	})

	return refName
}

func (r *repo) GetCompCommit(compDirName string) *object.Commit {
	return r.GetCommit(r.ComponentRepoSubpath(compDirName))
}

func (r *repo) GetRootCommit() string {
	if !r.IsClean() {
		log.Fatal("Error finding commit hash: uncommitted changes present")
	}
	head, err := r.gitRepo.Head()
	if err != nil {
		log.Fatal("Error opening head ref of git repo: %v", err)
	}
	return head.Hash().String()
}

func (r *repo) GetCommit(path string) *object.Commit {
	if !r.IsClean() {
		log.Fatal("Error finding commit hash: uncommitted changes present")
	}

	subPath := foxutils.Subpath(path, r.rootPath)
	iter, err := r.gitRepo.Log(&git.LogOptions{
		PathFilter: func(c string) bool {
			return strings.HasPrefix(c, subPath)
		},
	})
	if err != nil {
		log.Fatal("Error finding commit hash for path '%s': %v", subPath, err)
	}

	commit, err := iter.Next()
	if err != nil {
		log.Fatal("Error finding commit hash for path '%s': %v", subPath, err)
	}
	if commit == nil {
		log.Fatal("Error finding commit hash for path '%s': no commits have been made", subPath)
	}

	return commit
}

func (r *repo) ComponentsDir() string {
	return filepath.Join(r.appPath, "components")
}

func (r *repo) ComponentDir(comp string) string {
	return filepath.Join(r.ComponentsDir(), comp)
}

func (r *repo) ComponentRepoSubpath(comp string) string {
	return foxutils.Subpath(r.ComponentDir(comp), r.rootPath)
}

func (r *repo) ComponentAppSubpath(comp string) string {
	return foxutils.Subpath(r.ComponentDir(comp), r.appPath)
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
