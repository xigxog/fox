package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/repo"
)

var releaseCmd = &cobra.Command{
	Use:    "release [name]",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    release,
	Short:  "Release a deployed system",
	Long: `
The release command will activate the specified deployed KubeFox. This will 
trigger events matching routes of the system to be automatically sent to the 
deployed components. The system must be deployed to release it. Either id or 
tag must be provided.

Examples:
  #
  fox release --system demo/tag/v1.0.3 --environment dev/tag/v1.2
`,
}

func init() {
	releaseCmd.Flags().StringVarP(&cfg.Flags.Env, "env", "e", "", "Environment resource to release to (required)")
	releaseCmd.Flags().StringVarP(&cfg.Flags.EnvUID, "env-uid", "", "", "Environment resource UID to release to")
	releaseCmd.Flags().StringVarP(&cfg.Flags.EnvVersion, "env-version", "", "", "Environment resource version to release to")
	addCommonBuildFlags(releaseCmd)
	releaseCmd.MarkFlagRequired("env")

	rootCmd.AddCommand(releaseCmd)
}

func release(cmd *cobra.Command, args []string) {
	name := args[0]
	checkCommonDeployFlags(name)

	r := repo.New(cfg)
	rel := r.Release(name)
	// Makes output less cluttered.
	rel.ManagedFields = nil
	log.Marshal(rel)
}
