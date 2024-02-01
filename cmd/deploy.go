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
	"github.com/xigxog/kubefox/utils"
)

var deployCmd = &cobra.Command{
	Use:    "deploy",
	Args:   cobra.NoArgs,
	PreRun: setup,
	RunE:   runDeploy,
	Short:  "Deploy KubeFox App using the component code from the currently checked out Git commit",
	Long:   ``,
}

func init() {
	deployCmd.Flags().StringVarP(&cfg.Flags.AppDeployment, "name", "d", "", `name to use for AppDeployment, defaults to <APP NAME>-<VERSION | GIT REF | GIT COMMIT>`)
	deployCmd.Flags().StringVarP(&cfg.Flags.Version, "version", "s", "", "version to assign to the AppDeployment, making it immutable")
	deployCmd.Flags().BoolVarP(&cfg.Flags.CreateTag, "create-tag", "t", false, `create Git tag using the AppDeployment version`)
	addCommonDeployFlags(deployCmd)
	rootCmd.AddCommand(deployCmd)
}

func addCommonDeployFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cfg.Flags.Namespace, "namespace", "n", "", "namespace of KubeFox Platform")
	cmd.Flags().StringVarP(&cfg.Flags.Platform, "platform", "p", "", "name of KubeFox Platform to utilize")
	cmd.Flags().DurationVarP(&cfg.Flags.WaitTime, "wait", "", 0, "wait up the specified time for components to be ready")
	cmd.Flags().BoolVarP(&cfg.Flags.DryRun, "dry-run", "", false, "submit server-side request without persisting the resource")
}

func checkCommonDeployFlags() {
	if cfg.Flags.Platform != "" && cfg.Flags.Namespace == "" {
		log.Fatal("'namespace' flag required if 'platform' flag is provided.")
	}
	if cfg.Flags.CreateTag && cfg.Flags.Version == "" {
		log.Fatal("'version' flag required if 'create-tag' flag is set.")
	}
	if cfg.Flags.AppDeployment != "" && !utils.IsValidName(cfg.Flags.AppDeployment) {
		log.Fatal("Invalid resource name, valid names contain only lowercase alpha-numeric characters and dashes.")
	}
}

func runDeploy(cmd *cobra.Command, args []string) error {
	checkCommonDeployFlags()

	d := repo.New(cfg).Deploy(false)

	// Makes output less cluttered.
	d.Annotations = nil
	d.ManagedFields = nil

	log.Marshal(d)

	return nil
}
