package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xigxog/kubefox-cli/internal/config"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/kubefox"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	cfg = &config.Config{}
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
	rootCmd.PersistentFlags().StringVarP(&cfg.Flags.RepoPath, "repo", "r", pwd(), "path of git repo")
	rootCmd.PersistentFlags().StringVarP(&cfg.Flags.OutFormat, "output", "o", "yaml", `output format. One of: "json", "yaml"`)
	rootCmd.PersistentFlags().BoolVarP(&cfg.Flags.Info, "info", "i", true, "enable info output")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Flags.Verbose, "verbose", "v", false, "enable verbose output")
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
		log.Fatal("%v", err)
	}
}

func setup(cmd *cobra.Command, args []string) {
	log.OutputFormat = getOutFormat()
	log.EnableInfo = cfg.Flags.Info
	log.EnableVerbose = cfg.Flags.Verbose
	ctrl.SetLogger(logr.Logger{})

	cfg.Load()

	log.Verbose("gitCommit: %s, gitRef: %s", kubefox.GitCommit, kubefox.GitRef)
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
	case strings.EqualFold(cfg.Flags.OutFormat, "yaml") || strings.EqualFold(cfg.Flags.OutFormat, "yml"):
		return "yaml"
	case strings.EqualFold(cfg.Flags.OutFormat, "json"):
		return "json"
	case cfg.Flags.OutFormat == "":
		return "json"
	default:
		log.Fatal("Invalid output format '%s', provide one of: 'json', 'yaml'", cfg.Flags.OutFormat)
		return ""
	}
}
