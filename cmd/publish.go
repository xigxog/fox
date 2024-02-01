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
	"github.com/xigxog/fox/internal/repo"
)

var publishCmd = &cobra.Command{
	Use:    "publish",
	Args:   cobra.NoArgs,
	PreRun: setup,
	RunE:   runPublish,
	Short:  "Builds, pushes, and deploys KubeFox Apps using the component code from the currently checked out Git commit",
}

var (
	skipPush bool
)

func init() {
	publishCmd.Flags().StringVarP(&cfg.Flags.AppDeployment, "name", "d", "", `name to use for AppDeployment, defaults to <APP NAME>-<VERSION | GIT REF | GIT COMMIT>`)
	publishCmd.Flags().StringVarP(&cfg.Flags.Version, "version", "s", "", `version to assign to the AppDeployment, making it immutable`)
	publishCmd.Flags().BoolVarP(&cfg.Flags.CreateTag, "create-tag", "t", false, `create Git tag using the AppDeployment version`)
	publishCmd.Flags().BoolVarP(&skipPush, "skip-push", "", false, `do not push image after build`)
	publishCmd.Flags().BoolVarP(&cfg.Flags.SkipDeploy, "skip-deploy", "", false, `do not perform deployment after build`)
	addCommonBuildFlags(publishCmd)
	addCommonDeployFlags(publishCmd)

	rootCmd.AddCommand(publishCmd)
}

func runPublish(cmd *cobra.Command, args []string) error {
	if skipPush {
		cfg.Flags.PushImage = false
		cfg.Flags.SkipDeploy = cfg.Flags.Kind == "" && !cfg.Kind.AlwaysLoad
	} else {
		cfg.Flags.PushImage = true
	}
	if !cfg.Flags.SkipDeploy {
		checkCommonDeployFlags()
	}

	r := repo.New(cfg)
	d := r.Publish()
	// Makes output less cluttered.
	d.ManagedFields = nil
	log.Marshal(d)

	return nil
}
