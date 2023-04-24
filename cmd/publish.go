package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/repo"
	"github.com/xigxog/kubefox/libs/core/api/uri"
)

var publishCmd = &cobra.Command{
	Use:    "publish",
	Args:   cobra.NoArgs,
	PreRun: setup,
	Run:    publish,
	Short:  "The publish commands builds all components, publishes images, and adds the system to KubeFox.",
}

func init() {
	publishCmd.Flags().BoolVarP(&flags.Deploy, "deploy", "d", false, "deploy system after publishing")
	publishCmd.Flags().BoolVarP(&flags.SkipBuild, "skip-build", "s", false, "skip building of components")

	addCommonTagFlags(publishCmd)
	addCommonBuildFlags(publishCmd)

	rootCmd.AddCommand(publishCmd)
}

func publish(cmd *cobra.Command, args []string) {
	flags.PublishImage = true

	sURI, err := uri.New(cfg.GitHub.Org.Name, uri.System, flags.System)
	if err != nil {
		log.Fatal("Problem with system name: %v", err)
	}
	tURI, err := uri.New(cfg.GitHub.Org.Name, uri.System, flags.System, uri.Tag, flags.Tag)
	if err != nil {
		log.Fatal("Problem with tag name: %v", err)
	}

	repo := repo.New(cfg)
	sysObj := repo.GenerateSysObj()

	if !flags.SkipBuild {
		for _, app := range sysObj.Apps {
			for compName := range app.Components {
				log.Info("Building component '%s'", compName)
				repo.BuildComp(compName)
			}
		}
	}

	registerSystem()

	log.Info("Creating system object '%s'", sURI)
	log.Resp(admCli.Create(sURI, sysObj))

	log.Info("Creating system tag '%s'", tURI)
	log.Resp(admCli.Create(tURI, sysObj.GetId()))

	if flags.Deploy {
		log.Info("Deploying system tag '%s'", tURI)
		deploy(tURI)
	}

	log.Info("System successfully published")
}
