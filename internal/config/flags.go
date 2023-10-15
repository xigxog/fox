package config

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

	NoCache    bool
	PushImage  bool
	SkipDeploy bool
	ForceBuild bool
}
