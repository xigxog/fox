package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xigxog/kubefox-cli/internal/config"
	"github.com/xigxog/kubefox-cli/internal/log"
)

var (
	flags = config.Flags
	cfg   *config.Config
)

var rootCmd = &cobra.Command{
	Use:              "fox",
	PersistentPreRun: initViper,
	Short:            "CLI for interacting with KubeFox",
	Long: `
🦊 Fox is a CLI for interacting with KubeFox. You can use it to create, build, 
deploy, and release your KubeFox components.
`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flags.RepoPath, "repo", "r", pwd(), "path of git repo")
	rootCmd.PersistentFlags().StringVarP(&flags.OutFormat, "output", "o", "yaml", `output format. One of: "json", "yaml"`)
	rootCmd.PersistentFlags().BoolVarP(&flags.Verbose, "verbose", "v", false, "enable verbose output")
}

func initViper(cmd *cobra.Command, args []string) {
	viper.SetEnvPrefix("fox")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.BindPFlags(cmd.Flags())
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Value.String() == f.DefValue && viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			cmd.Flags().Set(f.Name, viper.GetString(f.Name))
		}
	})
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal("Error running command: %v", err)
	}
}

func setup(cmd *cobra.Command, args []string) {
	log.Setup(getOutFormat(), flags.Verbose)
	cfg = config.Load()
}

func pwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting working dir: %v", err)
	}

	return filepath.Clean(wd)
}

func getOutFormat() string {
	switch {
	case strings.EqualFold(flags.OutFormat, "yaml") || strings.EqualFold(flags.OutFormat, "yml"):
		return "yaml"
	case strings.EqualFold(flags.OutFormat, "json"):
		return "json"
	case flags.OutFormat == "":
		return "json"
	default:
		log.Fatal("Invalid output format '%s', provide one of: 'json', 'yaml'", flags.OutFormat)
		return ""
	}
}
