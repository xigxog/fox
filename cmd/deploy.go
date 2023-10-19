package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/repo"
	"github.com/xigxog/kubefox-cli/internal/utils"
)

var deployCmd = &cobra.Command{
	Use:    "deploy [name]",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    runDeploy,
	Short:  "Deploy components to the KubeFox platform",
	Long:   ``,
}

func init() {
	addCommonDeployFlags(deployCmd)
	rootCmd.AddCommand(deployCmd)
}

func addCommonDeployFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cfg.Flags.Platform, "platform", "p", "", "Platform to run Components with")
	cmd.Flags().StringVarP(&cfg.Flags.Namespace, "namespace", "n", "", "Namespace of Platform")
}

func checkCommonDeployFlags(name string) {
	if cfg.Flags.Platform != "" && cfg.Flags.Namespace == "" {
		log.Fatal("'namespace' flag required if 'platform' flag is provided.")
	}
	if name != utils.Clean(name) {
		log.Fatal("Invalid resource name, valid names contain only lowercase alpha-numeric characters and dashes.")
	}
}

func runDeploy(cmd *cobra.Command, args []string) {
	name := args[0]
	checkCommonDeployFlags(name)

	r := repo.New(cfg)
	d := r.Deploy(name)
	// Makes output less cluttered.
	d.ManagedFields = nil
	log.InfoNewline()
	log.Marshal(d)
}
