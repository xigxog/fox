package config

var Flags = &flags{}

type flags struct {
	// persistent flags defined in root command
	RepoPath  string
	OutFormat string

	Verbose bool

	// flags used by  subcommands
	Builder    string
	Env        string
	EnvUID     string
	EnvVersion string
	Namespace  string
	Platform   string
	Deployment string
	Kind       string

	PublishImage bool
	ClearCache   bool
}
