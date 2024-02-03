// Copyright 2023 XigXog
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"runtime/debug"

	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/log"
)

// XXX Update this before making release. This is hardcoded to ensure that
// the correct version is shown when Fox is setup using `go install`.
const version = "v0.8.0"

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
