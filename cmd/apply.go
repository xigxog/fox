package cmd

import (
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/api/admin"
	"github.com/xigxog/kubefox/libs/core/api/uri"

	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:    "apply -f <filename>",
	Args:   cobra.NoArgs,
	PreRun: setup,
	Run:    apply,
	Short:  "Create or update a KubeFox resource from a file",
	Long: `
This command will create or update the provided KubeFox resource from the 
provided file. If the "tag" flag is passed the resource will be tagged after 
update.
`,
}

func init() {
	applyCmd.Flags().StringVarP(&flags.Filename, "filename", "f", "", "file that contains the KubeFox object (required)")
	applyCmd.Flags().StringVarP(&flags.Tag, "tag", "t", "", "tag name, use of semantic versioning is recommended")
	applyCmd.MarkFlagRequired("filename")

	rootCmd.AddCommand(applyCmd)
}

func apply(cmd *cobra.Command, args []string) {
	u, obj := getObjFromFile(flags.Filename)
	log.Resp(admCli.Apply(u, obj))

	if flags.Tag != "" {
		if aObj, ok := obj.(admin.Object); ok {
			tagURI, err := uri.New(cfg.GitHub.Org.Name, aObj.GetKind(), aObj.GetName(), uri.Id, aObj.GetId())
			if err != nil {
				log.Fatal("Error creating tag: %v", err)
			}
			tagObj(tagURI)

		} else {
			log.Fatal("Only configs, environments, and systems can be tagged")
		}
	}
}
