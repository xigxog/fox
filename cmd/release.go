package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/repo"
)

var releaseCmd = &cobra.Command{
	Use:    "release (NAME | COMMIT | SHORT COMMIT | VERSION | TAG | BRANCH)",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    release,
	Short:  "Release specified AppDeployment and VirtualEnvironment",
	Long: `
The release command activates the routes of the components belonging to the 
specified AppDeployment. This causes genesis events matching components' routes
to be automatically sent to the component with the specified environment being 
injected.

Examples:

    # Release the AppDeployment named 'main' using the 'dev' Virtual Environment.
    fox release main --virtual-env dev

    # Release the AppDeployment with version 'v1.2.3' using the 'prod' 
	# VirtualEnvironment, creating an VirtualEnvironmentSnapshot if needed.
    fox release v1.2.3 --virtual-env prod --create-snapshot
`,
}

func init() {
	releaseCmd.Flags().StringVarP(&cfg.Flags.VirtEnv, "virtual-env", "e", "", "name of ClusterVirtualEnvironment, VirtualEnvironment, or VirtualEnvironmentSnapshot to use")
	releaseCmd.Flags().BoolVarP(&cfg.Flags.CreateVirtEnv, "create-snapshot", "c", false, "create an immutable snapshot of environment and use for release")

	addCommonDeployFlags(releaseCmd)

	releaseCmd.MarkFlagRequired("virtual-env")

	rootCmd.AddCommand(releaseCmd)
}

func release(cmd *cobra.Command, args []string) {
	appDep := args[0]
	checkCommonDeployFlags(appDep)

	rel := repo.New(cfg).Release(appDep)

	// Makes output less cluttered.
	rel.ManagedFields = nil
	log.Marshal(rel)
}
