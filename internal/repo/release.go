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
	"github.com/xigxog/kubefox/api/kubernetes/v1alpha1"
	"github.com/xigxog/kubefox/core"
	"github.com/xigxog/kubefox/k8s"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *repo) Release(appDepId string) *v1alpha1.VirtualEnv {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	platform := r.k8s.GetPlatform()

	appDep, err := r.findAppDep(ctx, platform, appDepId)
	if err != nil {
		log.Fatal("Error finding AppDeployment: %v", err)
	}

	var envSnapName string
	envSnap := &v1alpha1.VirtualEnvSnapshot{}
	err = r.k8s.Get(ctx, k8s.Key(platform.Namespace, r.cfg.Flags.VirtEnv), envSnap)
	if k8s.IgnoreNotFound(err) != nil {
		log.Fatal("Error getting VirtualEnvSnapshot: %v", err)
	}

	if k8s.IsNotFound(err) {
		if envSnap, err = r.k8s.SnapshotVirtualEnv(ctx, platform.Namespace, r.cfg.Flags.VirtEnv); err != nil {
			log.Fatal("Error getting VirtualEnv: %v", err)
		}
	} else {
		envSnapName = envSnap.Name
	}

	problems, err := appDep.Validate(envSnap.Data, func(name string, typ api.ComponentType) (api.Adapter, error) {
		switch typ {
		case api.ComponentTypeHTTPAdapter:
			a := &v1alpha1.HTTPAdapter{}
			if err := r.k8s.Get(ctx, k8s.Key(appDep.Namespace, name), a); err != nil {
				return nil, err
			}
			return a, nil

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

	if envSnapName == "" && r.cfg.Flags.CreateVirtEnv {
		if envSnapName, err = r.createEnvSnapshot(ctx, envSnap); err != nil {
			log.Fatal("Error creating VirtualEnvSnapshot: %v", err)
		}
	}

	env := &v1alpha1.VirtualEnv{
		TypeMeta: v1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       "VirtualEnv",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      envSnap.Spec.Source.Name,
			Namespace: platform.Namespace,
		},
		Spec: v1alpha1.VirtualEnvSpec{
			Release: &v1alpha1.Release{
				AppDeployment: v1alpha1.ReleaseAppDeployment{
					Name:    appDep.Name,
					Version: appDep.Spec.Version,
				},
				VirtualEnvSnapshot: envSnapName,
			},
		},
	}
	if err := r.k8s.Apply(ctx, env); err != nil {
		log.Fatal("Error updating VirtualEnv with Release: %v", err)
	}

	r.waitForReady(platform, &appDep.Spec)

	return env
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
			label: appDepId,
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

func (r *repo) createEnvSnapshot(ctx context.Context, env *v1alpha1.VirtualEnvSnapshot) (string, error) {
	log.Verbose("checking for existing VirtualEnvSnapshot of VirtualEnv '%s' with resourceVersion '%s'",
		env.Spec.Source.Name, env.Spec.Source.ResourceVersion)

	list := &v1alpha1.VirtualEnvSnapshotList{}
	if err := r.k8s.List(ctx, list, client.MatchingLabels{
		api.LabelK8sVirtualEnv:            env.Name,
		api.LabelK8sSourceResourceVersion: env.ResourceVersion,
	}); err != nil {
		return "", err
	}
	for _, s := range list.Items {
		// Double check source is equal.
		if s.Spec.Source == env.Spec.Source {
			log.Verbose("found existing snapshot '%s'", list.Items[0].Name)
			return s.Name, nil
		}
	}

	log.VerboseMarshal(env, "creating VirtualEnvSnapshot")
	if err := r.k8s.Create(ctx, env); err != nil {
		return "", err
	}

	return env.Name, nil
}
