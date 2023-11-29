package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/repo"
	"github.com/xigxog/kubefox/utils"
)

var deployCmd = &cobra.Command{
	Use:    "deploy (NAME)",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    runDeploy,
	Short:  "Deploy KubeFox App using the version from the currently checked out Git commit",
	Long:   ``,
}

func init() {
	addCommonDeployFlags(deployCmd)
	rootCmd.AddCommand(deployCmd)
}

func addCommonDeployFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cfg.Flags.Namespace, "namespace", "n", "", "namespace of KubeFox Platform")
	cmd.Flags().StringVarP(&cfg.Flags.Platform, "platform", "p", "", "name of KubeFox Platform to utilize")
	cmd.Flags().DurationVarP(&cfg.Flags.WaitTime, "wait", "", 0, "wait up the specified time for components to be ready")
	cmd.Flags().BoolVarP(&cfg.Flags.DryRun, "dry-run", "", false, "submit server-side request without persisting the resource")
}

func checkCommonDeployFlags(name string) {
	if cfg.Flags.Platform != "" && cfg.Flags.Namespace == "" {
		log.Fatal("'namespace' flag required if 'platform' flag is provided.")
	}
	if !utils.IsValidName(name) {
		log.Fatal("Invalid resource name, valid names contain only lowercase alpha-numeric characters and dashes.")
	}
}

func runDeploy(cmd *cobra.Command, args []string) {
	name := args[0]
	checkCommonDeployFlags(name)

	d := repo.New(cfg).Deploy(name, false)

	// Makes output less cluttered.
	d.ManagedFields = nil
	log.Marshal(d)
}
