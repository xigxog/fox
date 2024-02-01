// Copyright 2023 XigXog
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package repo

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/utils"
	"github.com/xigxog/kubefox/api"
	common "github.com/xigxog/kubefox/api/kubernetes"
	"github.com/xigxog/kubefox/api/kubernetes/v1alpha1"
	"github.com/xigxog/kubefox/core"
	"github.com/xigxog/kubefox/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *repo) Release(appDepId string) *v1alpha1.VirtualEnvironment {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	platform := r.k8s.GetPlatform()

	appDep, err := r.findAppDep(ctx, platform, appDepId)
	if err != nil {
		log.Fatal("Error finding AppDeployment: %v", err)
	}
	ve := &v1alpha1.VirtualEnvironment{}
	if err := r.k8s.Get(ctx, k8s.Key(platform.Namespace, r.cfg.Flags.VirtEnv), ve); err != nil {
		log.Fatal("Error getting VirtualEnvironment: %v", err)
	}
	env := &v1alpha1.Environment{}
	if err := r.k8s.Get(ctx, k8s.Key("", ve.Spec.Environment), env); err != nil {
		log.Fatal("Error getting Environment: %v", err)
	}
	ve.Data.Import(&env.Data)

	problems, err := appDep.Validate(&ve.Data,
		func(name string, typ api.ComponentType) (common.Adapter, error) {
			switch typ {
			case api.ComponentTypeHTTPAdapter:
				a := &v1alpha1.HTTPAdapter{}
				return a, r.k8s.Get(ctx, k8s.Key(appDep.Namespace, name), a)

			default:
				return nil, core.ErrNotFound()
			}
		})
	if err != nil {
		log.Fatal("Error validating Release: %v", err)
	}
	if len(problems) > 0 {
		log.InfoMarshal(problems, "Release problems:")
		if !utils.YesNoPrompt("Problems that would prevent Release activation exist, continue?", false) {
			log.Fatal("Release aborted.")
		}
	}

	origVE := ve.DeepCopy()
	if ve.Spec.Release == nil {
		ve.Spec.Release = &v1alpha1.Release{}
	}

	if ve.Spec.Release.Apps == nil {
		ve.Spec.Release.Apps = map[string]v1alpha1.ReleaseApp{}
	}
	ve.Spec.Release.Apps[appDep.Spec.AppName] = v1alpha1.ReleaseApp{
		AppDeployment: appDep.Name,
		Version:       appDep.Spec.Version,
	}

	if err := r.k8s.Merge(ctx, ve, origVE); err != nil {
		log.Fatal("Error updating Release: %v", err)
	}

	r.waitForReady(platform, &appDep.Spec)

	return ve
}

func (r *repo) findAppDep(ctx context.Context, platform *v1alpha1.Platform, appDepId string) (*v1alpha1.AppDeployment, error) {
	// Try getting AppDeployment by name.
	appDep := &v1alpha1.AppDeployment{}
	if err := r.k8s.Get(ctx, k8s.Key(platform.Namespace, appDepId), appDep); k8s.IgnoreNotFound(err) != nil {
		return nil, err
	} else if err == nil {
		return appDep, nil
	}

	// If AppDeployment not found by name search using the following labels
	// listed in order of precedence.
	labels := []string{
		api.LabelK8sAppCommit,
		api.LabelK8sAppCommitShort,
		api.LabelK8sAppVersion,
		api.LabelK8sAppTag,
		api.LabelK8sAppBranch,
	}
	for _, label := range labels {
		appDepList := &v1alpha1.AppDeploymentList{}
		if err := r.k8s.List(ctx, appDepList, client.MatchingLabels{
			api.LabelK8sAppName: r.app.Name,
			label:               appDepId,
		}); err != nil {
			return nil, err
		}

		switch l := len(appDepList.Items); {
		case l == 1:
			return &appDepList.Items[0], nil
		case l > 1:
			log.Info("Found %d matching AppDeployments.", l)
			return r.pickAppDep(appDepList), nil
		}
	}

	return nil, core.ErrNotFound()
}

func (r *repo) pickAppDep(appDepList *v1alpha1.AppDeploymentList) *v1alpha1.AppDeployment {
	for i, appDep := range appDepList.Items {
		log.Printf("%d. %s/%s\n", i+1, appDep.Namespace, appDep.Name)
	}

	var input string
	log.Printf("Select the KubeFox AppDeployment to use: ")
	fmt.Scanln(&input)
	i, err := strconv.Atoi(input)
	if err != nil {
		return r.pickAppDep(appDepList)
	}
	i = i - 1
	if i < 0 || i >= len(appDepList.Items) {
		return r.pickAppDep(appDepList)
	}

	selected := &appDepList.Items[i]
	log.InfoNewline()

	return selected
}
