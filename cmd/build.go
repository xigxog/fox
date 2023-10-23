package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/repo"
)

var buildCmd = &cobra.Command{
	Use:    "build (component name)",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    build,
	Short:  "Build and optionally push an OCI image of component",
	Long: `
The build command will use Docker to build the specified component. By default
components are built using a KubeFox defined Dockerfile. A custom Dockerfile can
be provided my placing it in the root directory of the component. Please note
that the build working directory is the root of the repository, not the
component directory.

Examples:

    # Build and push OCI image for my-component.
    fox build my-component --publish
`,
}

func init() {
	buildCmd.Flags().BoolVarP(&cfg.Flags.PushImage, "push", "", false, "publish image to OCI image registry")
	addCommonBuildFlags(buildCmd)

	rootCmd.AddCommand(buildCmd)
}

func addCommonBuildFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cfg.Flags.Kind, "kind", "k", "", "if provided the built image will be loaded into the Kind cluster")
	cmd.Flags().BoolVarP(&cfg.Flags.NoCache, "no-cache", "", false, "do not use cache when building image")
	cmd.Flags().BoolVarP(&cfg.Flags.ForceBuild, "force", "", false, "force build even if component image exists")
}

func build(cmd *cobra.Command, args []string) {
	r := repo.New(cfg)
	r.Build(args[0])
}
