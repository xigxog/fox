package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/repo"
)

var releaseCmd = &cobra.Command{
	Use:    "release [release name]",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    release,
	Short:  "Release app using the version of the currently checked out Git commit",
	Long: `
The release command will ensure all components are deployed and then activate 
their routes. This causes genesis events matching component's routes to be 
automatically sent to the component with the specified environment being 
injected.

Examples:
  # Create a release named 'staging' using the 'qa' environment.
  fox release staging --env qa
`,
}

func init() {
	releaseCmd.Flags().StringVarP(&cfg.Flags.Env, "env", "e", "", "environment resource to release to (required)")
	releaseCmd.Flags().StringVarP(&cfg.Flags.EnvUID, "env-uid", "", "", "environment resource UID to release to")
	releaseCmd.Flags().StringVarP(&cfg.Flags.EnvVersion, "env-version", "", "", "environment resource version to release to")
	addCommonBuildFlags(releaseCmd)
	addCommonDeployFlags(releaseCmd)
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
