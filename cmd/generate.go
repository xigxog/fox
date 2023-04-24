package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/repo"
)

var genCmd = &cobra.Command{
	Use:    "generate",
	Args:   cobra.NoArgs,
	PreRun: setup,
	Run:    genSysObj,
	Short:  "Generates a system object from a KubeFox managed git repo",
}

func init() {
	rootCmd.AddCommand(genCmd)
}

func genSysObj(cmd *cobra.Command, args []string) {
	sysObj := repo.New(cfg).GenerateSysObj()
	log.Marshal(sysObj)
}
