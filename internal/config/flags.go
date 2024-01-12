// Copyright 2023 XigXog
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package config

import "time"

type Flags struct {
	// persistent flags defined in root command
	AppPath   string
	OutFormat string

	DryRun  bool
	Info    bool
	Verbose bool

	// flags used by subcommands
	AppDeployment   string
	Builder         string
	Kind            string
	Namespace       string
	Platform        string
	Version         string
	VirtEnv         string
	RegistryAddress string
	RegistryToken   string

	CreateTag  bool
	ForceBuild bool
	NoCache    bool
	PushImage  bool
	Quickstart bool
	SkipDeploy bool

	WaitTime time.Duration
}
