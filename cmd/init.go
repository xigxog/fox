package cmd

import (
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/xigxog/kubefox-cli/efs"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/repo"
	"github.com/xigxog/kubefox-cli/internal/utils"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:    "init",
	Short:  "Initialize a KubeFox System",
	Args:   cobra.NoArgs,
	PreRun: setup,
	Run:    initSystem,
	Long: `Initialize a KubeFox system. This command will:

  - Ping the Kubefox platform to ensure a valid installation and endpoint
  - Write a demo KubeFox system in the directory provided
  - Register the system with the KubeFox platform
`,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func initSystem(cmd *cobra.Command, args []string) {
	pingKubeFox()
	initDemo()
	registerSystem()

	log.Info("KubeFox system initialization complete")
}

func pingKubeFox() {
	log.Info("Checking connectivity to Kubefox Platform at '%s'", flags.URL)
	log.Resp(admCli.Ping())
}

func initDemo() {
	dir := flags.SysRepoPath
	log.Info("Writing files for a demo KubeFox system to '%s'", dir)
	utils.EnsureDir(dir)
	if !utils.IsDirEmpty(dir) {
		log.Fatal("Directory '%s' is not empty", dir)
	}

	fs.WalkDir(efs.EFS, efs.DemoSystemPath,
		func(efsPath string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Fatal("Error initializing KubeFox system: %v", err)
			}
			if d.IsDir() {
				return nil
			}

			// Go will not embed directories containing a go.mod file. To
			// resolve this the extension .trim is added. Removing it here.
			path := strings.TrimPrefix(strings.TrimSuffix(efsPath, ".trim"), efs.DemoSystemPath)
			path = filepath.Join(dir, path)

			log.Verbose("Writing file '%s'", path)
			utils.EnsureDirForFile(path)
			if utils.FileExists(path) {
				log.Fatal("File '%s' exists", path)
			}

			data, _ := efs.EFS.ReadFile(efsPath)
			if err := os.WriteFile(path, data, 0666); err != nil {
				log.Fatal("Error creating file: %v", err)
			}

			return nil
		})

	nr, err := git.PlainInit(dir, false)
	if err != nil {
		log.Fatal("Error initializing git repo: %v", err)
	}
	nr.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{
			fmt.Sprintf("https://github.com/%s/%s.git", cfg.GitHub.Org.Name, flags.System),
		},
	})

	r := repo.New(cfg)
	r.CommitAll("Initial commit of KubeFox system. This is where it all begins...")
}
