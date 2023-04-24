package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/api/uri"
)

var deleteCmd = &cobra.Command{
	Use:    "delete <resource path>",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    deleteResource,
	Short:  "Delete KubeFox resources",
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func deleteResource(cmd *cobra.Command, args []string) {
	u := getResURI(args)
	if u.SubKind() != uri.Tag && u.SubKind() != uri.Deployment && u.SubKind() != uri.Release {
		log.Fatal("Only tags, deployments, and releases can be deleted")
	}

	log.Resp(admCli.Delete(u))
}
