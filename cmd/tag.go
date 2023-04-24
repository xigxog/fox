package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/api/uri"
)

var tagCmd = &cobra.Command{
	Use:    "tag <resource path> --tag <tag name>",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    tagObjCmd,
	Short:  "Tag a KubeFox object",
	Long: `
Use the tag command to label a KubeFox object by id. It creates a new resource
to access the object at {kind}/{name}/tag/{tag name}.

Examples:
  # Tags the dev environment creating the resource environment/dev/tag/v1.0
  fox tag env/dev/id/630b9bb6-ae5a-46ec-8b23-bee91044de6f --tag v1.0

  # Tags the demo system creating the resource system/demo/tag/v1.0.3
  fox tag system/dev/id/8cc42b131fbb96e8aa298d62bc09a0941b836626 --tag v1.0.3
`,
}

func init() {
	addCommonTagFlags(tagCmd)
	rootCmd.AddCommand(tagCmd)
}
func addCommonTagFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&flags.Tag, "tag", "t", "", "tag name, use of semantic versioning is recommended (required)")
	cmd.MarkFlagRequired("tag")
}

func tagObjCmd(cmd *cobra.Command, args []string) {
	tagObj(getResURI(args))
}

func tagObj(srcURI uri.URI) {
	if srcURI.Kind() != uri.Config && srcURI.Kind() != uri.Environment && srcURI.Kind() != uri.System {
		log.Fatal("Only configs, environments, and systems can be tagged")
	}
	if srcURI.SubKind() != uri.Id {
		log.Fatal("Tags can only be made using an id resource")
	}

	tagURI, err := uri.New(cfg.GitHub.Org.Name, srcURI.Kind(), srcURI.Name(), uri.Tag, flags.Tag)
	if err != nil {
		log.Fatal("Error tagging object: %v", err)
	}
	log.VerboseResp(admCli.Create(tagURI, srcURI.SubPath()))
	log.Info("Tag resource '%s' created", tagURI)
}
