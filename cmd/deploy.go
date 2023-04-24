package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/api/admin/v1alpha1"
	"github.com/xigxog/kubefox/libs/core/api/maker"
	"github.com/xigxog/kubefox/libs/core/api/uri"
)

var deployCmd = &cobra.Command{
	Use:    "deploy <system path>",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    deployRun,
	Short:  "Deploy a system to the KubeFox platform",
	Long: `
The deploy command runs the components of the specified system on the Kubefox
platform.

Examples:
  # Deploys the components of the demo system with the tag v1.0.3
  fox deploy system/demo/tag/v1.0.3

  # Deploys the components of the demo system with the id 8cc42b131fbb96e8aa298d62bc09a0941b836626
  fox deploy system/demo/id/8cc42b131fbb96e8aa298d62bc09a0941b836626

  # Deploys the components of the latest version of the demo system
  fox deploy system/demo
`,
}

func init() {
	rootCmd.AddCommand(deployCmd)
}

func deployRun(cmd *cobra.Command, args []string) {
	deploy(getResURI(args))
}

func deploy(sysURI uri.URI) {
	if sysURI.Kind() != uri.System {
		log.Fatal("Only systems can be deployed")
	}

	deploy := maker.Empty[v1alpha1.Deployment]()
	deploy.System = string(sysURI.Key())

	u, err := uri.New(cfg.GitHub.Org.Name, uri.Platform, cfg.KubeFox.Platform, uri.Deployment)
	if err != nil {
		log.Fatal("Error creating URI for deployment: %v", err)
	}

	registerSystem()
	log.Resp(admCli.Create(u, deploy))
}
