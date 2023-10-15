/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/kubefox"
)

var verCmd = &cobra.Command{
	Use:   "version",
	Run:   runVer,
	Short: "Show version information of Fox CLI",
}

func init() {
	rootCmd.AddCommand(verCmd)
}

func runVer(cmd *cobra.Command, args []string) {
	log.Marshal(map[string]any{
		"gitCommit": kubefox.GitCommit,
		"gitRef":    kubefox.GitRef,
	})
}
