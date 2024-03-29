// Copyright 2023 XigXog
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/log"
)

var cfgCmd = &cobra.Command{
	Use:   "config",
	Args:  cobra.NoArgs,
	Short: "Configure 🦊 Fox",
	Long: `
Use the config subcommand to help setup your local environment.
`,
}

var cfgShowCmd = &cobra.Command{
	Use:    "show",
	Args:   cobra.NoArgs,
	PreRun: setup,
	Run: func(cmd *cobra.Command, args []string) {
		log.Marshal(cfg)
	},
	Short: "Show the current configuration",
}

var cfgSetupCmd = &cobra.Command{
	Use:    "setup",
	Args:   cobra.NoArgs,
	PreRun: setup,
	Run: func(cmd *cobra.Command, args []string) {
		if !cfg.Fresh {
			cfg.Setup()
		}
	},
	Short: "Run setup to configure 🦊 Fox",
}

func init() {
	rootCmd.AddCommand(cfgCmd)

	cfgCmd.AddCommand(cfgShowCmd)
	cfgCmd.AddCommand(cfgSetupCmd)
}
