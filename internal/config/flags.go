package config

import "time"

type Flags struct {
	// persistent flags defined in root command
	RepoPath  string
	OutFormat string

	Info    bool
	Verbose bool

	// flags used by subcommands
	Builder    string
	Env        string
	EnvUID     string
	EnvVersion string
	Namespace  string
	Platform   string
	Kind       string

	WaitTime time.Duration

	NoCache    bool
	PushImage  bool
	SkipDeploy bool
	ForceBuild bool
}
