package cmd

import (
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/config"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/admin"
)

var flags = config.Flags

var (
	admCli admin.Client
	cfg    *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "fox",
	Short: "CLI for interacting with KubeFox",
	Long: `
🦊 Fox is a CLI for interacting with KubeFox. You can use it to create, build, 
validate, deploy, and release your KubeFox components, apps, and systems.
`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flags.SysRepoPath, "system-repo", "r", pwd(), "path of the system git repo")
	rootCmd.PersistentFlags().StringVarP(&flags.URL, "url", "u", "", "url to the KubeFox API")
	rootCmd.PersistentFlags().StringVarP(&flags.OutFormat, "output", "o", "yaml", `output format. One of: "json", "yaml"`)
	rootCmd.PersistentFlags().BoolVarP(&flags.Verbose, "verbose", "v", false, "enable verbose output")
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

	if flags.System == "" {
		flags.System = filepath.Base(flags.SysRepoPath)
	}
	if flags.URL == "" {
		flags.URL = cfg.KubeFox.URL
	}

	admCli = admin.NewClient(admin.ClientConfig{
		URL:      flags.URL,
		Timeout:  30 * time.Second,
		Insecure: true,
		Log:      log.Logger(),
	})
}
