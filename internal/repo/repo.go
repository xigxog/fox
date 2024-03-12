// Copyright 2023 XigXog
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package repo

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
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

	gitRepo *git.Repository
	k8s     *kubernetes.Client
	docker  *docker.Client

	ctx    context.Context
	cancel context.CancelFunc
}

type App struct {
	Title             string `json:"title,omitempty"`
	Description       string `json:"description,omitempty"`
	Name              string `json:"name"`
	ContainerRegistry string `json:"containerRegistry,omitempty"`
}

func New(cfg *config.Config) *repo {
	cfg.CleanPaths(false)

	if !strings.HasPrefix(cfg.AppPath, cfg.RepoPath) {
		log.Fatal("The app is not part of the Git repo.")
	}

	app, err := ReadApp(cfg.AppPath)
	if err != nil {
		log.Fatal("Error reading the repo's 'app.yaml', try running 'fox init': %v", err)
	}

	log.Verbose("Opening git repo '%s'", cfg.RepoPath)
	gitRepo, err := git.PlainOpen(cfg.RepoPath)
	if err != nil {
		log.Fatal("Error opening git repo '%s': %v", cfg.RepoPath, err)
	}

	d, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal("Error creating Docker client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Flags.Timeout)

	return &repo{
		cfg:     cfg,
		app:     app,
		gitRepo: gitRepo,
		k8s:     kubernetes.NewClient(cfg),
		docker:  d,
		ctx:     ctx,
		cancel:  cancel,
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
	if filepath.Base(r.GetTagRef()) == "tag" {
		log.Info("Tag '%s' for commot '%s' exists.", tag)
	}

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
	hash := r.GetCompHash(compDirName)
	return r.GetCompImage(name, hash)
}

func (r *repo) GetCompImage(name, hash string) string {
	return fmt.Sprintf("%s/%s/%s:%s", r.cfg.GetContainerRegistry().Address, r.app.Name, name, hash)
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

func (r *repo) GetCompHash(compDirName string) string {
	h := md5.New()

	err := filepath.Walk(r.ComponentDir(compDirName),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				// compPath := foxutils.Subpath(path, r.cfg.RepoPath)
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()

				if _, err := io.Copy(h, f); err != nil {
					return err
				}
			}

			return nil
		})
	if err != nil {
		log.Fatal("Error generating Component hash: %v", err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func (r *repo) GetCommit() *object.Commit {
	if !r.IsClean() {
		log.Fatal("Error finding commit hash: uncommitted changes present")
	}
	head, err := r.gitRepo.Head()
	if err != nil {
		log.Fatal("Error opening head ref of git repo: %v", err)
	}

	c, err := r.gitRepo.CommitObject(head.Hash())
	if err != nil {
		log.Fatal("Error getting commit '%s' for head ref of git repo: %v", head.Hash().String(), err)
	}

	return c
}

func (r *repo) AppYAMLBuildSubpath() string {
	return foxutils.Subpath(filepath.Join(r.cfg.AppPath, "app.yaml"), r.cfg.RepoPath)
}

func (r *repo) ComponentBuildSubpath(compDirName string) string {
	return foxutils.Subpath(r.ComponentDir(compDirName), r.cfg.RepoPath)
}

func (r *repo) ComponentsDir() string {
	return filepath.Join(r.cfg.AppPath, "components")
}

func (r *repo) ComponentDir(comp string) string {
	return filepath.Join(r.ComponentsDir(), comp)
}

func (r *repo) ComponentRepoSubpath(comp string) string {
	return foxutils.Subpath(r.ComponentDir(comp), r.cfg.RepoPath)
}

func (r *repo) IsClean() bool {
	if os.Getenv("FOX_DRAGON_IGNORE_UNCOMMITTED") == "true" {
		return true
	}

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
