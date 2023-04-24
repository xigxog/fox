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
	buildCmd.Flags().BoolVarP(&flags.PublishImage, "publish", "i", false, "publish image to OCI image registry")
	addCommonBuildFlags(buildCmd)

	rootCmd.AddCommand(buildCmd)
}

func addCommonBuildFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&flags.Registry, "registry", "g", "ghcr.io", "OCI image registry to publish images to")
	cmd.Flags().StringVarP(&flags.Builder, "builder", "b", "paketobuildpacks/builder:base", "BuildPack builder to use")
	cmd.Flags().BoolVarP(&flags.ClearCache, "clear-cache", "c", false, `clear BuildPack cache`)
}

func build(cmd *cobra.Command, args []string) {
	r := repo.New(cfg)
	r.BuildComp(args[0])
}
