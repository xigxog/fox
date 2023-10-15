package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/repo"
)

var buildCmd = &cobra.Command{
	Use:    "build [component name]",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    build,
	Short:  "Build and publish an OCI image of your component",
	Long: `
This will use the specified Cloud Native BuildPack to build, package, and optionally publish the component provided as an OCI image to the registry.

Examples:
  # Build and push container image for my-component
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
	r.BuildComp(args[0])
}
