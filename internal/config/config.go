package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cli/oauth/device"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/utils"
	"github.com/xigxog/kubefox/libs/core/admin"
	"github.com/xigxog/kubefox/libs/core/api/uri"
	"github.com/xigxog/kubefox/libs/core/validator"
	"sigs.k8s.io/yaml"
)

const clientId = "c19364e47478c620b1b3"

type Config struct {
	GitHub  GitHub  `json:"github" validate:"required"`
	KubeFox KubeFox `json:"kubefox" validate:"required"`

	path string
}

type GitHub struct {
	Org   GitHubOrg  `json:"org" validate:"required"`
	User  GitHubUser `json:"user" validate:"required"`
	Token string     `json:"token" validate:"required"`
}

type GitHubUser struct {
	Id        int    `json:"id" validate:"required"`
	Name      string `json:"login" validate:"required"`
	AvatarURL string `json:"avatar_url" validate:"required,url"`
	URL       string `json:"html_url" validate:"required,url"`
}

type GitHubOrg struct {
	Id   int    `json:"id" validate:"required"`
	Name string `json:"login" validate:"required"`
	URL  string `json:"url" validate:"required,url"`
}

type GitHubError struct {
	Msg    string `json:"message"`
	DocURL string `json:"documentation_url"`
}

type KubeFox struct {
	URL      string `json:"url" validate:"required,url"`
	Platform string `json:"platform" validate:"required"`
}

func Load() *Config {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Error accessing user's home directory: %v", err)
	}
	path := filepath.Join(home, ".config/kubefox/config.yaml")

	log.Verbose("Loading Kubefox config from '%s'", path)

	cfg := &Config{path: path}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		cfg.Setup()
	} else if err != nil {
		log.Fatal("Error reading KubeFox config file: %v", err)
	} else if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Fatal("Error unmarshaling KubeFox config: %v", err)
	}

	v := validator.New(log.Logger())
	if errs := v.Validate(cfg); errs != nil {
		log.VerboseMarshal(errs, "Config is not valid")
		cfg.Setup()
	}

	return cfg
}

func (cfg *Config) Setup() {
	// write is called after each step so values are not lost on next run
	cfg.setupGitHub()
	cfg.Write()

	cfg.setupKubeFox()
	cfg.Write()
}

func (cfg *Config) Write() {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatal("Error marshaling KubeFox config: %v", err)
	}

	utils.EnsureDirForFile(cfg.path)
	if err := os.WriteFile(cfg.path, b, 0600); err != nil {
		log.Fatal("Error writing KubeFox config file: %v", err)
	}
}

func (cfg *Config) setupGitHub() {
	httpCl := http.DefaultClient
	scopes := []string{"read:user", "read:org", "read:packages", "write:packages"}
	code, err := device.RequestCode(httpCl, "https://github.com/login/device/code", clientId, scopes)
	if err != nil {
		log.Fatal("%v", err)
	}

	log.Info("Fox needs to create a secret to push and pull container images to GitHub Packages.")
	log.Info("Copy this code '%s', then open '%s' in your browser.", code.UserCode, code.VerificationURI)

	accToken, err := device.Wait(context.Background(), httpCl, "https://github.com/login/oauth/access_token",
		device.WaitOptions{
			ClientID:   clientId,
			DeviceCode: code,
		})
	if err != nil {
		log.Fatal("%v", err)
	}
	cfg.GitHub.Token = accToken.Token

	orgs := []*GitHubOrg{}
	callGitHub("GET", "https://api.github.com/user/orgs", cfg.GitHub.Token, &orgs)
	callGitHub("GET", "https://api.github.com/user", cfg.GitHub.Token, &cfg.GitHub.User)

	switch len(orgs) {
	case 0:
		log.Fatal("A GitHub organization is required, please set one up. https://bit.ly/3mNYkh1")
	case 1:
		cfg.GitHub.Org = *orgs[0]
	default:
		cfg.GitHub.Org = *pickOrg(orgs)
	}
	cfg.GitHub.Org.Name = strings.ToLower(cfg.GitHub.Org.Name)
}

func (cfg *Config) setupKubeFox() {
	// make sure we have needed GitHub config before proceeding
	v := validator.New(log.Logger())
	if errs := v.Validate(cfg.GitHub); errs != nil {
		cfg.setupGitHub()
	}

	def := "https://127.0.0.1:30443"
	if cfg.KubeFox.URL != "" {
		def = cfg.KubeFox.URL
	}
	var input string
	fmt.Printf("Enter the URL of the KubeFox API (default '%s'): ", def)
	fmt.Scanln(&input)
	if input == "" {
		input = def
	}

	u, err := url.Parse(input)
	if err != nil {
		log.Error("Invalid URL: %v, please try again", err)
		cfg.setupKubeFox()
	} else {
		cfg.KubeFox.URL = u.String()
	}

	admCl := admin.NewClient(admin.ClientConfig{
		URL:      cfg.KubeFox.URL,
		Insecure: true,
		Log:      log.Logger(),
	})

	listURI, _ := uri.New(cfg.GitHub.Org.Name, uri.Platform)
	resp, err := admCl.List(listURI)
	if err != nil {
		log.Fatal("Error communicating with KubeFox API: %v", err)
	}

	list, ok := resp.Data.([]any)
	if !ok {
		log.Fatal("Unexpected response from KubeFox API")
	}
	if len(list) != 1 {
		log.Fatal("Unexpected response from KubeFox API")
	}
	cfg.KubeFox.Platform = fmt.Sprintf("%s", list[0])
}

func pickOrg(orgs []*GitHubOrg) *GitHubOrg {
	for i, o := range orgs {
		fmt.Printf("%d. %s\n", i+1, o.Name)
	}
	var input string
	fmt.Printf("Select the GitHub org to use (default 1): ")
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

func callGitHub(verb, url, token string, body any) {
	req, err := http.NewRequest(verb, url, nil)
	if err != nil {
		log.Fatal("Error calling GitHub: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
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
