package config

var Flags = &flags{}

type flags struct {
	// persistent flags defined in root command
	SysRepoPath string
	URL         string
	OutFormat   string

	Verbose bool

	// flags used by  subcommands
	Tag     string
	Builder string

	Msg      string
	Filename string
	Registry string
	Config   string
	Env      string
	System   string

	Deploy       bool
	SkipDeploy   bool
	SkipBuild    bool
	PublishImage bool
	ClearCache   bool
}
