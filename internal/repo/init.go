package repo

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	gitcfg "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/xigxog/kubefox-cli/efs"
	"github.com/xigxog/kubefox-cli/internal/config"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/utils"
	"sigs.k8s.io/yaml"
)

func Init(cfg *config.Config) {
	repoPath := cfg.Flags.RepoPath

	app, err := ReadApp(repoPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Error("An app definition already exists but appears to be invalid: %v.", err)
		if !utils.YesNoPrompt("Would you like to reinitialize the repo?", true) {
			return
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		log.VerboseMarshal(app, "App definition:")
		log.Info("A valid app definition already exists.")
		if utils.YesNoPrompt("Would you like to add a component?", true) {
			addComponent(repoPath)
		}
		initGit(repoPath, app, cfg)
		return
	}

	app = &App{}
	log.Info("Let's initialize a KubeFox repo!")

	log.InfoNewline()
	log.Info("To get things started quickly Fox can create a hello-world app which includes")
	log.Info("two components and example environments for testing.")
	if utils.YesNoPrompt("Would you like to initialize the hello-world app?", false) {
		initDir(efs.HelloWorldPath, repoPath)
		initGit(repoPath, app, cfg)
		return
	}

	log.InfoNewline()
	log.Info("Fox needs to create an app definition for the repo. The definition is stored in")
	log.Info("the 'app.yaml' file in the root of the repo. The first thing it needs is a name")
	log.Info("for the app. The name is used as part of Kubernetes resource names so it must")
	log.Info("contain only lowercase alpha-numeric characters and dashes. But don't worry")
	log.Info("you can enter a more human friendly title and description.")
	app.Name = utils.NamePrompt("app", utils.Clean(repoPath), true)
	app.Title = utils.InputPrompt("Enter the app's title", "", false)
	app.Description = utils.InputPrompt("Enter the app's description", "", false)

	appPath := filepath.Join(repoPath, "app.yaml")
	b, err := yaml.Marshal(app)
	if err != nil {
		log.Fatal("Error marshaling app definition: %v", err)
	}
	utils.EnsureDirForFile(appPath)
	if err := os.WriteFile(appPath, b, 0644); err != nil {
		log.Fatal("Error writing app definition file: %v", err)
	}
	utils.EnsureDir(filepath.Join(repoPath, ComponentsDirName))
	utils.EnsureDir(filepath.Join(repoPath, "hack"))
	utils.EnsureDir(filepath.Join(repoPath, "libs"))

	if utils.YesNoPrompt("Would you like to add a component?", true) {
		addComponent(repoPath)
	}
	initGit(repoPath, app, cfg)
}

func initGit(repoPath string, app *App, cfg *config.Config) {
	wt := osfs.New(repoPath)
	dot, _ := wt.Chroot(git.GitDirName)
	s := filesystem.NewStorage(dot, cache.NewObjectLRUDefault())
	nr, err := git.InitWithOptions(s, wt, git.InitOptions{DefaultBranch: plumbing.Main})
	alreadyExists := errors.Is(err, git.ErrRepositoryAlreadyExists)
	if err != nil && !alreadyExists {
		log.Fatal("Error initializing git repo: %v", err)
	}

	if !alreadyExists {
		if cfg.GitHub.Org.Name != "" {
			nr.CreateRemote(&gitcfg.RemoteConfig{
				Name: "origin",
				URLs: []string{
					fmt.Sprintf("https://github.com/%s/%s.git", cfg.GitHub.Org.Name, app.Name),
				},
			})
		}
		r := New(cfg)
		r.CommitAll("And so it begins...")
	}

	log.InfoNewline()
	log.Info("KubeFox repo initialization complete!")
}

func initDir(in, out string) {
	log.Verbose("Writing files from EFS '%s' to '%s", in, out)

	utils.EnsureDir(out)
	fs.WalkDir(efs.EFS, in,
		func(efsPath string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Fatal("Error initializing repo: %v", err)
			}
			if d.IsDir() {
				return nil
			}

			// Go will not embed directories containing a go.mod file. To
			// resolve this the extension '.trim' is added. Removing it here.
			path := strings.TrimPrefix(strings.TrimSuffix(efsPath, ".trim"), in)
			path = filepath.Join(out, path)

			log.Verbose("Writing file '%s'", path)
			utils.EnsureDirForFile(path)
			if utils.FileExists(path) {
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

func addComponent(dir string) {
	name := utils.NamePrompt("component", "", true)
	initDir(efs.ComponentPath, filepath.Join(dir, ComponentsDirName, name))

	if utils.YesNoPrompt("Would you like to add another component?", false) {
		addComponent(dir)
	}
}
