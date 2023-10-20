package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
)

var cfgCmd = &cobra.Command{
	Use:              "config",
	Args:             cobra.NoArgs,
	PersistentPreRun: setup,
	Short:            "Configure 🦊 Fox",
	Long: `
Use the config subcommand to help setup your local environment.
`,
}

var cfgShowCmd = &cobra.Command{
	Use:  "show",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		log.Marshal(cfg)
	},
	Short: "Show the current configuration",
}

var cfgSetupCmd = &cobra.Command{
	Use:  "setup",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if !cfg.Fresh {
			cfg.Setup()
		}
	},
	Short: "Run setup to configure 🦊 Fox",
}

func init() {
	rootCmd.AddCommand(cfgCmd)

	cfgCmd.AddCommand(cfgShowCmd)
	cfgCmd.AddCommand(cfgSetupCmd)
}
