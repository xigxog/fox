package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cli/oauth/device"
	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/utils"
	"sigs.k8s.io/yaml"
)

const (
	LocalRegistry  = "localhost/kubefox"
	GitHubClientId = "a76b4dc61b6fec162ef6"
)

type Config struct {
	GitHub            GitHub            `json:"github"`
	KubeFox           KubeFox           `json:"kubefox"`
	Kind              Kind              `json:"kind"`
	ContainerRegistry ContainerRegistry `json:"containerRegistry"`

	Flags Flags `json:"-"`
	Fresh bool  `json:"-"`

	path string
}

type GitHub struct {
	Org   GitHubOrg  `json:"org"`
	User  GitHubUser `json:"user"`
	Token string     `json:"token"`
}

type GitHubUser struct {
	Id        int    `json:"id"`
	Name      string `json:"login"`
	AvatarURL string `json:"avatar_url" validate:"url"`
	URL       string `json:"html_url" validate:"url"`
}

type GitHubOrg struct {
	Id   int    `json:"id"`
	Name string `json:"login"`
	URL  string `json:"url" validate:"url"`
}

type GitHubError struct {
	Msg    string `json:"message"`
	DocURL string `json:"documentation_url"`
}

type KubeFox struct {
	Namespace string `json:"namespace"`
	Platform  string `json:"platform"`
}

type Kind struct {
	ClusterName string `json:"clusterName"`
	AlwaysLoad  bool   `json:"alwaysLoad"`
}

type ContainerRegistry struct {
	Address string `json:"address" validate:"required"`
	Token   string `json:"token"`
}

func (cfg *Config) IsRegistryLocal() bool {
	return strings.HasPrefix(cfg.ContainerRegistry.Address, LocalRegistry)
}

func (cfg *Config) Load() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Error accessing user's home directory: %v", err)
	}
	cfg.path = filepath.Join(home, ".config/kubefox/config.yaml")

	log.Verbose("Loading Kubefox config from '%s'", cfg.path)

	b, err := os.ReadFile(cfg.path)
	if errors.Is(err, fs.ErrNotExist) {
		log.Info("It looks like this is the first time you are using 🦊 Fox. Welcome!")
		log.InfoNewline()
		log.Info("🦊 Fox needs some information from you to configure itself. The setup process only")
		log.Info("needs to be run once, but if you ever want change things you can use the")
		log.Info("command 'fox config setup'.")
		log.InfoNewline()

		cfg.Setup()
	} else if err != nil {
		log.Fatal("Error reading KubeFox config file '%s': %v", err, cfg.path)
	}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		log.Fatal("Error unmarshaling KubeFox config '%s': %v", err, cfg.path)
	}
	if cfg.ContainerRegistry.Address == "" {
		log.Info("It looks like the container registry is missing from your config. Rerunning")
		log.Info("setup to fix the issue.")
		log.InfoNewline()

		cfg.Setup()
	}
}

func (cfg *Config) CleanPaths(defAppToWd bool) {
	var err error

	if cfg.Flags.RepoPath == "" {
		cfg.Flags.RepoPath = utils.Find(".git", utils.Wd(), string(filepath.Separator))
	}
	cfg.Flags.RepoPath, err = filepath.Abs(cfg.Flags.RepoPath)
	if err != nil {
		log.Fatal("Unable to resolve repo path: %v", err)
	}

	if cfg.Flags.AppPath == "" {
		if defAppToWd {
			cfg.Flags.AppPath = utils.Wd()
		} else {
			cfg.Flags.AppPath = utils.Find("app.yaml", utils.Wd(), cfg.Flags.RepoPath)
		}
	}
	cfg.Flags.AppPath, err = filepath.Abs(cfg.Flags.AppPath)
	if err != nil {
		log.Fatal("Unable to resolve app path: %v", err)
	}

	log.Verbose("Repo path: %s", cfg.Flags.RepoPath)
	log.Verbose("App path: %s", cfg.Flags.AppPath)

	if cfg.Flags.AppPath == "" {
		log.Fatal("Could not find app definition (app.yaml).")
	}
	if !strings.HasPrefix(cfg.Flags.AppPath, cfg.Flags.RepoPath) {
		log.Fatal("The app is not part of the Git repo.")
	}
}

func (cfg *Config) Setup() {
	log.Info("Please make sure your workstation has Docker installed (https://docs.docker.com/engine/install)")
	log.Info("and that KubeFox is installed (https://docs.kubefox.io/install) on your Kubernetes cluster.")
	log.InfoNewline()
	log.Info("If you don't have a Kubernetes cluster you can run one locally with kind (https://kind.sigs.k8s.io)")
	log.Info("to experiment with KubeFox.")
	log.InfoNewline()
	log.Info("🦊 Fox needs a place to store the component images it will build, normally this is")
	log.Info("a remote container registry. However, if you only want to use KubeFox locally")
	log.Info("with kind you can skip this step.")
	kindOnly := utils.YesNoPrompt("Are you only using KubeFox with local kind cluster?", false)
	if kindOnly {
		cfg.ContainerRegistry.Address = LocalRegistry
		cfg.ContainerRegistry.Token = ""
		cfg.Kind.ClusterName = utils.NamePrompt("kind cluster", "kind", true)
		cfg.Kind.AlwaysLoad = true
		log.InfoNewline()
		cfg.done()
		return
	}
	log.InfoNewline()
	log.Info("Great! If you don't already have a container registry 🦊 Fox can help setup the")
	log.Info("GitHub container registry (ghcr.io).")
	useGH := utils.YesNoPrompt("Would you like to use ghcr.io?", true)
	log.InfoNewline()
	if useGH {
		cfg.setupGitHub()
	} else {
		log.Info("No problem. 🦊 Fox just needs to know which container registry to use. Please be")
		log.Info("sure you have permissions to pull and push images to the registry.")
		cfg.ContainerRegistry.Address = utils.InputPrompt("Enter the container registry you'd like to use", "", true)
		cfg.ContainerRegistry.Token = utils.InputPrompt("Enter the container registry access token", "", false)
	}
	log.InfoNewline()

	cfg.done()
}

func (cfg *Config) done() {
	cfg.Fresh = true
	cfg.Write()
	log.InfoNewline()
	log.Info("Congrats, you are ready to use KubeFox!")
	log.Info("Check out the quickstart for next steps (https://docs.kubefox.io/quickstart/).")
	log.Info("If you run into any problems please let us know on GitHub (https://github.com/xigxog/kubefox/issues).")
}

func (cfg *Config) setupGitHub() {
	log.Info("🦊 Fox needs to create two access tokens. The first is used by 🦊 Fox and is only")
	log.Info("stored locally. It allows 🦊 Fox to read your GitHub user and organizations and to")
	log.Info("push and pull container images to ghcr.io. This information never leaves your")
	log.Info("workstation.")
	log.InfoNewline()
	log.Info("The second access token is used by Kubernetes to pull component images from")
	log.Info("ghcr.io. It is stored locally and as a Secret on your Kubernetes cluster.")
	log.InfoNewline()

	log.Info("This will create the access token for 🦊 Fox.")
	cfg.GitHub.Token = getToken([]string{"read:user", "read:org", "read:packages", "write:packages"})
	log.InfoNewline()
	log.Info("Next, this will create the access token for Kubernetes to pull images.")
	cfg.ContainerRegistry.Token = getToken([]string{"read:packages"})
	log.InfoNewline()

	orgs := []*GitHubOrg{}
	cfg.callGitHub("GET", "https://api.github.com/user/orgs", &orgs)
	cfg.callGitHub("GET", "https://api.github.com/user", &cfg.GitHub.User)

	switch len(orgs) {
	case 0:
		log.Error("Oh no, a GitHub organization is required to use GitHub container registry,")
		log.Fatal("please create one (https://bit.ly/3mNYkh1) before continuing.")
	case 1:
		cfg.GitHub.Org = *orgs[0]
		log.InfoNewline()
	default:
		cfg.GitHub.Org = *pickOrg(orgs)
	}
	cfg.ContainerRegistry.Address = fmt.Sprintf("ghcr.io/%s", cfg.GitHub.Org.Name)
}

func getToken(scopes []string) string {
	code, err := device.RequestCode(http.DefaultClient, "https://github.com/login/device/code", GitHubClientId, scopes)
	if err != nil {
		log.Fatal("%v", err)
	}
	log.Printf("Copy this code '%s', then open '%s' in your browser.\n", code.UserCode, code.VerificationURI)
	accToken, err := device.Wait(context.Background(), http.DefaultClient, "https://github.com/login/oauth/access_token",
		device.WaitOptions{
			ClientID:   GitHubClientId,
			DeviceCode: code,
		})
	if err != nil {
		log.Fatal("%v", err)
	}

	return accToken.Token
}

func pickOrg(orgs []*GitHubOrg) *GitHubOrg {
	for i, o := range orgs {
		log.Printf("%d. %s\n", i+1, o.Name)
	}
	var input string
	log.Printf("Select the GitHub organization to use (default 1): ")
	fmt.Scanln(&input)
	if input == "" {
		input = "1"
	}
	i, err := strconv.Atoi(input)
	if err != nil {
		return pickOrg(orgs)
	}
	i = i - 1
	if i < 0 || i >= len(orgs) {
		return pickOrg(orgs)
	}

	return orgs[i]
}

func (cfg *Config) Write() {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatal("Error marshaling KubeFox config: %v", err)
	}

	log.VerboseMarshal(cfg, "config:")

	utils.EnsureDirForFile(cfg.path)
	if err := os.WriteFile(cfg.path, b, 0600); err != nil {
		log.Fatal("Error writing KubeFox config file: %v", err)
	}
	log.Info("Configuration successfully written to '%s'.", cfg.path)
}

func (cfg *Config) callGitHub(verb, url string, body any) {
	req, err := http.NewRequest(verb, url, nil)
	if err != nil {
		log.Fatal("Error calling GitHub: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+cfg.GitHub.Token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode >= 400 && err == nil {
		ghErr := GitHubError{}
		if dErr := json.NewDecoder(resp.Body).Decode(&ghErr); dErr != nil {
			err = dErr
		} else {
			err = errors.New(ghErr.Msg)
		}
	}
	if err != nil {
		log.Fatal("Error calling GitHub: %v", err)
	}
	if err := json.NewDecoder(resp.Body).Decode(body); err != nil {
		log.Fatal("Error calling GitHub: %v", err)
	}
}
