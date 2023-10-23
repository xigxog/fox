package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/repo"
)

var publishCmd = &cobra.Command{
	Use:    "publish (deploy-name)",
	Args:   cobra.MaximumNArgs(1),
	PreRun: setup,
	RunE:   runPublish,
	Short:  "Builds, pushes, and deploys KubeFox apps using the version of the currently checked out Git commit",
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

	var name string
	if !cfg.Flags.SkipDeploy {
		name = args[0]
		checkCommonDeployFlags(name)
	}

	r := repo.New(cfg)
	d := r.Publish(name)
	// Makes output less cluttered.
	d.ManagedFields = nil
	log.Marshal(d)

	return nil
}
