package cmd

import (
	_ "embed"

	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/repo"
)

var initCmd = &cobra.Command{
	Use:    "init",
	Short:  "Initialize a KubeFox repo",
	Args:   cobra.NoArgs,
	PreRun: setup,
	Run:    initRepo,
	Long: `
The init command create the skelton of a KubeFox repo with sample components in 
the provided dir.
`,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func initRepo(cmd *cobra.Command, args []string) {
	repo.Init(cfg)
}
