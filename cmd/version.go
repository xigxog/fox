/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"runtime/debug"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/log"
)

var verCmd = &cobra.Command{
	Use:   "version",
	Run:   runVer,
	Short: "Show version information of 🦊 Fox",
}

func init() {
	rootCmd.AddCommand(verCmd)
}

func runVer(cmd *cobra.Command, args []string) {
	var (
		version, commit, date string
		modified              bool = true
	)
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				commit = s.Value
			case "vcs.time":
				date = s.Value
			case "vcs.modified":
				modified, _ = strconv.ParseBool(s.Value)
			}
		}
	}

	if modified {
		log.Verbose("binary built from source with uncommitted changes")
	}
	log.Marshal(map[string]any{
		"version": version,
		"commit":  commit,
		"date":    date,
	})
}
