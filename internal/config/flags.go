package config

import "time"

type Flags struct {
	// persistent flags defined in root command
	RepoPath  string
	AppPath   string
	OutFormat string

	DryRun  bool
	Info    bool
	Verbose bool

	// flags used by subcommands
	Builder       string
	AppDeployment string
	VirtEnv       string

	Kind      string
	Namespace string
	Platform  string

	WaitTime time.Duration

	NoCache       bool
	PushImage     bool
	SkipDeploy    bool
	ForceBuild    bool
	CreateVirtEnv bool
}
