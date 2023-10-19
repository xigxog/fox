package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/repo"
)

var publishCmd = &cobra.Command{
	Use:    "publish [deploy-name]",
	Args:   cobra.MaximumNArgs(1),
	PreRun: setup,
	RunE:   runPublish,
	Short:  "The publish commands builds all components, publishes images, and deploys them to KubeFox.",
}

var (
	skipPush bool
)

func init() {
	publishCmd.Flags().BoolVarP(&skipPush, "skip-push", "", false, `do not push image after build`)
	publishCmd.Flags().BoolVarP(&cfg.Flags.SkipDeploy, "skip-deploy", "", false, `do not perform deployment after build`)
	addCommonBuildFlags(publishCmd)
	addCommonDeployFlags(publishCmd)

	rootCmd.AddCommand(publishCmd)
}

func runPublish(cmd *cobra.Command, args []string) error {
	if skipPush {
		cfg.Flags.PushImage = false
		cfg.Flags.SkipDeploy = cfg.Flags.Kind == "" && !cfg.Kind.AlwaysLoad
	} else {
		cfg.Flags.PushImage = true
	}
	if !cfg.Flags.SkipDeploy && len(args) == 0 {
		return fmt.Errorf("accepts 1 arg(s), received 0")
	}
	if !cfg.Flags.SkipDeploy {
		checkCommonDeployFlags(args[0])
	}

	r := repo.New(cfg)
	r.Publish()

	if !cfg.Flags.SkipDeploy {
		d := r.Deploy(args[0])
		// Makes output less cluttered.
		d.ManagedFields = nil
		log.InfoNewline()
		log.Marshal(d)
	}

	return nil
}
