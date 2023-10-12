package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/repo"
)

var publishCmd = &cobra.Command{
	Use:    "publish",
	Args:   cobra.NoArgs,
	PreRun: setup,
	Run:    runPublish,
	Short:  "The publish commands builds all components, publishes images, and optionally deploys them to KubeFox.",
}

func init() {
	publishCmd.Flags().StringVarP(&flags.Deployment, "deploy", "d", "", "create deployment after publish is complete")
	addCommonBuildFlags(publishCmd)
	addCommonDeployFlags(publishCmd)

	rootCmd.AddCommand(publishCmd)
}

func runPublish(cmd *cobra.Command, args []string) {
	dName := flags.Deployment
	if dName != "" {
		checkCommonDeployFlags(dName)
	}

	r := repo.New(cfg)
	r.Publish()

	if dName != "" {
		d := r.Deploy(dName)
		// Makes output less cluttered.
		d.ManagedFields = nil
		log.Marshal(d)
	}
}
