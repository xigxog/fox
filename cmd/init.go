// Copyright 2023 XigXog
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	_ "embed"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/repo"
)

var initCmd = &cobra.Command{
	Use:    "init",
	Short:  "Initialize a KubeFox App",
	Args:   cobra.NoArgs,
	PreRun: setup,
	Run:    initRepo,
	Long: strings.TrimSpace(`
The init command creates the skelton of a KubeFox App and ensures a Git 
repository is present. It will optionally create simple 'hello-world' app to get
you started.
`),
}

func init() {
	initCmd.Flags().BoolVarP(&cfg.Flags.Quickstart, "quickstart", "", false, `use defaults to setup KubeFox for quickstart guide`)
	rootCmd.AddCommand(initCmd)
}

func initRepo(cmd *cobra.Command, args []string) {
	repo.Init(cfg)
}
