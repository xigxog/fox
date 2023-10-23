package config

import "time"

type Flags struct {
	// persistent flags defined in root command
	RepoPath  string
	AppPath   string
	OutFormat string

	Info    bool
	Verbose bool

	// flags used by subcommands
	Builder    string
	Deployment string
	Env        string
	EnvUID     string
	EnvVersion string
	Kind       string
	Namespace  string
	Platform   string

	WaitTime time.Duration

	NoCache    bool
	PushImage  bool
	SkipDeploy bool
	ForceBuild bool
}
