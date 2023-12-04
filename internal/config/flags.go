package config

import "time"

type Flags struct {
	// persistent flags defined in root command
	AppPath   string
	OutFormat string
	RepoPath  string

	DryRun  bool
	Info    bool
	Verbose bool

	// flags used by subcommands
	AppDeployment string
	Builder       string
	Kind          string
	Namespace     string
	Platform      string
	Version       string
	VirtEnv       string

	CreateVirtEnv bool
	CreateTag     bool
	ForceBuild    bool
	NoCache       bool
	PushImage     bool
	Quickstart    bool
	SkipDeploy    bool

	WaitTime time.Duration
}
