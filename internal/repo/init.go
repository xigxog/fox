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
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	gitcfg "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/xigxog/fox/efs"
	"github.com/xigxog/fox/internal/config"
	"github.com/xigxog/fox/internal/kubernetes"
	"github.com/xigxog/fox/internal/log"
	foxutils "github.com/xigxog/fox/internal/utils"
	"github.com/xigxog/kubefox/utils"
)

func Init(cfg *config.Config) {
	initApp(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Flags.Timeout)
	defer cancel()

	if cfg.Flags.Quickstart {
		c := kubernetes.NewClient(cfg)
		p := c.CreatePlatform(ctx, "kubefox-demo", "demo")
		c.WaitPlatformReady(time.Minute*5, p, nil)

		log.Info("KubeFox initialized for the quickstart guide!")

	} else {
		log.InfoNewline()
		// Creates new platform if none exist.
		kubernetes.NewClient(cfg).GetPlatform(ctx)
		log.Info("KubeFox App initialization complete!")
	}
}

func initApp(cfg *config.Config) {
	cfg.CleanPaths(true)

	if cfg.Flags.Quickstart {
		initDir(efs.HelloWorldPath, cfg.AppPath)
		initGit(cfg.RepoPath, cfg)
		return
	}

	app, err := ReadApp(cfg.AppPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Error("An KubeFox App definition already exists but appears to be invalid: %v.", err)
		if !foxutils.YesNoPrompt("Would you like to reinitialize the app?", true) {
			return
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		log.VerboseMarshal(app, "App definition:")
		log.Info("A valid KubeFox App definition already exists.")
		initGit(cfg.RepoPath, cfg)
		return
	}

	app = &App{}
	log.Info("Let's initialize a KubeFox App!")
	log.InfoNewline()
	log.Info("To get things started quickly 🦊 Fox can create a 'hello-world' KubeFox App which")
	log.Info("includes two components and example environments for testing.")
	if foxutils.YesNoPrompt("Would you like to initialize the 'hello-world' KubeFox App?", false) {
		initDir(efs.HelloWorldPath, cfg.AppPath)
		initGit(cfg.RepoPath, cfg)
		return
	}
	log.InfoNewline()
	log.Info("🦊 Fox needs to create an KubeFox App definition. The definition is stored in the")
	log.Info("'app.yaml' file in the root of the repo. The first thing it needs is a name for")
	log.Info("the app. The name is used as part of Kubernetes resource names so it must")
	log.Info("contain only lowercase alpha-numeric characters and dashes. But don't worry you")
	log.Info("can enter a more human friendly title and description.")
	app.Name = foxutils.NamePrompt("KubeFox App", utils.CleanName(cfg.AppPath), true)
	app.Title = foxutils.InputPrompt("Enter the KubeFox App's title", "", false)
	app.Description = foxutils.InputPrompt("Enter the KubeFox App's description", "", false)

	WriteApp(cfg.AppPath, app)
	initGit(cfg.RepoPath, cfg)
}

func initGit(repoPath string, cfg *config.Config) {
	wt := osfs.New(repoPath)
	dot, _ := wt.Chroot(git.GitDirName)
	s := filesystem.NewStorage(dot, cache.NewObjectLRUDefault())
	nr, err := git.InitWithOptions(s, wt, git.InitOptions{DefaultBranch: plumbing.Main})
	alreadyExists := errors.Is(err, git.ErrRepositoryAlreadyExists)
	if err != nil && !alreadyExists {
		log.Fatal("Error initializing git repo: %v", err)
	}

	r := New(cfg)
	foxutils.EnsureDir(r.ComponentsDir())

	if !alreadyExists {
		var remoteURL string
		if cfg.GitHub.Org.Name != "" {
			remoteURL = fmt.Sprintf("https://github.com/%s/%s.git", cfg.GitHub.Org.Name, filepath.Base(repoPath))
		}

		if !cfg.Flags.Quickstart {
			remoteURL = foxutils.InputPrompt("Enter URL for remote Git repo", remoteURL, false)
			if remoteURL != "" {
				_, err := nr.CreateRemote(&gitcfg.RemoteConfig{
					Name: "origin",
					URLs: []string{remoteURL},
				})
				if err != nil {
					log.Warn("Unable to set remote Git repo: %v", err)
				}
			}
		}

		r.CommitAll("And so it begins...")
	}
}

func initDir(in, out string) {
	log.Verbose("Writing files from EFS '%s' to '%s", in, out)

	foxutils.EnsureDir(out)
	fs.WalkDir(efs.EFS, in,
		func(efsPath string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Fatal("Error initializing app: %v", err)
			}
			if d.IsDir() {
				return nil
			}

			// Go will not embed directories containing a go.mod file. To
			// resolve this the extension '.trim' is added. Removing it here.
			path := strings.TrimPrefix(strings.TrimSuffix(efsPath, ".trim"), in)
			path = filepath.Join(out, path)

			log.Verbose("Writing file '%s'", path)
			foxutils.EnsureDirForFile(path)
			if foxutils.FileExists(path) {
				log.Verbose("File '%s' exists, skipping...", path)
				return nil
			}

			data, _ := efs.EFS.ReadFile(efsPath)
			if err := os.WriteFile(path, data, 0644); err != nil {
				log.Fatal("Error creating file: %v", err)
			}

			return nil
		})
}
