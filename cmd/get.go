package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/api/uri"
)

var getCmd = &cobra.Command{
	Use:    "get <resource path>",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    getResource,
	Short:  "Get KubeFox resource by path",
	Long: `
This command will print the output of a Kubefox resource by default.
If a path to a resource type is given, this command will list all Kubefox resources of this type.

All kubefox objects are available for this command.
	
You must use the full path to resources, as represented in the API Documentation:
- https://github.com/kubefox/controller/blob/alpha1/README.md

Example:
To list all available environments of your kubefox platform you would run:
	fox get environment

To get the spec of your kubefox dev environment you would run:
	fox get environment/dev

To list the tags available to your kubefox dev environment you would run:
	fox get environment/dev/tag	
	`,
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func getResource(cmd *cobra.Command, args []string) {
	u := getResURI(args)
	if u.Name() == "" || u.SubPath() == "" {
		log.Resp(admCli.List(u))
	} else if u.SubKind() == uri.Deployment || u.SubKind() == uri.Release {
		log.Resp(admCli.Get(u, newSubResource(u)))
	} else {
		log.Resp(admCli.Get(u, newResource(u)))
	}
}
