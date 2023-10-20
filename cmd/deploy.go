package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/repo"
	"github.com/xigxog/fox/internal/utils"
)

var deployCmd = &cobra.Command{
	Use:    "deploy [deployment name]",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    runDeploy,
	Short:  "Deploy KubeFox app using the version from the currently checked out Git commit",
	Long:   ``,
}

func init() {
	addCommonDeployFlags(deployCmd)
	rootCmd.AddCommand(deployCmd)
}

func addCommonDeployFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cfg.Flags.Platform, "platform", "p", "", "platform to run components with")
	cmd.Flags().StringVarP(&cfg.Flags.Namespace, "namespace", "n", "", "namespace of platform")
	cmd.Flags().DurationVarP(&cfg.Flags.WaitTime, "wait", "", 0, "wait up the specified time for components to be ready")
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
	log.Marshal(d)
}
