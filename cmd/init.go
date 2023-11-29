package cmd

import (
	_ "embed"

	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/repo"
)

var initCmd = &cobra.Command{
	Use:    "init",
	Short:  "Initialize a KubeFox App",
	Args:   cobra.NoArgs,
	PreRun: setup,
	Run:    initRepo,
	Long: `
The init command creates the skelton of a KubeFox App and ensures a Git 
repository is present. It will optionally create simple 'hello-world' app to get
you started.
`,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func initRepo(cmd *cobra.Command, args []string) {
	repo.Init(cfg)
}
