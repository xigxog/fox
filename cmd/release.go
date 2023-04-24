package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/api/admin/v1alpha1"
	"github.com/xigxog/kubefox/libs/core/api/maker"
	"github.com/xigxog/kubefox/libs/core/api/uri"
)

var releaseCmd = &cobra.Command{
	Use:    "release",
	Args:   cobra.NoArgs,
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
	releaseCmd.Flags().StringVar(&flags.System, "system", "", "System to release (required)")
	releaseCmd.Flags().StringVarP(&flags.Env, "environment", "e", "", "Environment to release to (required)")

	releaseCmd.MarkFlagRequired("system")
	releaseCmd.MarkFlagRequired("environment")

	rootCmd.AddCommand(releaseCmd)
}

func release(cmd *cobra.Command, args []string) {
	sysURI, err := uri.New(cfg.GitHub.Org.Name, uri.System, flags.System)
	if err != nil {
		log.Fatal("Error creating system URI: %v", err)
	}
	envURI, err := uri.New(cfg.GitHub.Org.Name, uri.Environment, flags.Env)
	if err != nil {
		log.Fatal("Error creating environment URI: %v", err)
	}

	release := maker.Empty[v1alpha1.Release]()
	release.System = string(sysURI.Key())
	release.Environment = string(envURI.Key())
	u, err := uri.New(cfg.GitHub.Org.Name, uri.Platform, cfg.KubeFox.Platform, uri.Release)
	if err != nil {
		log.Fatal("Error creating release URI: %v", err)
	}

	registerSystem()
	log.Resp(admCli.Create(u, release))
}
