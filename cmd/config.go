package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
)

var cfgCmd = &cobra.Command{
	Use:              "config",
	Args:             cobra.NoArgs,
	PersistentPreRun: setup,
	Short:            "Configure your KubeFox Client",
	Long: `
Use the config subcommand to help setup your local CLI environment.
`,
}

var cfgShowCmd = &cobra.Command{
	Use:  "show",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		log.Marshal(cfg)
	},
	Short: "Show your current KubeFox configuration",
}

var cfgSetupCmd = &cobra.Command{
	Use:  "setup",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if !cfg.Fresh {
			cfg.Setup()
		}
	},
	Short: "Update your current KubeFox configuration",
}

func init() {
	rootCmd.AddCommand(cfgCmd)

	cfgCmd.AddCommand(cfgShowCmd)
	cfgCmd.AddCommand(cfgSetupCmd)
}
