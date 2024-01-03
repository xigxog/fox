package cmd

import (
	"runtime/debug"

	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/log"
)

// XXX Update this before making release. This is hardcoded to ensure that
// the correct version is shown when Fox is setup using `go install`.
const version = "v0.7.0-alpha"

type BuildInfo struct {
	Version   string `json:"version,omitempty"`
	Commit    string `json:"commit,omitempty"`
	BuildDate string `json:"buildDate,omitempty"`
}

var verCmd = &cobra.Command{
	Use:   "version",
	Run:   runVer,
	Short: "Show version information of 🦊 Fox",
}

func init() {
	rootCmd.AddCommand(verCmd)
}

func runVer(cmd *cobra.Command, args []string) {
	var commit, buildDate string
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				commit = s.Value
			case "vcs.time":
				buildDate = s.Value
			}
		}
	}

	log.Marshal(&BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
	})
}
