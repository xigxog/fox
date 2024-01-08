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
	origVE := ve.DeepCopy()

	env := &v1alpha1.Environment{}
	if err := r.k8s.Get(ctx, k8s.Key("", ve.Spec.Environment), env); err != nil {
		log.Fatal("Error getting Environment: %v", err)
	}
	ve.Merge(env)

	problems, err := appDep.Validate(&ve.Data, func(name string, typ api.ComponentType) (api.Adapter, error) {
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

	snapName := r.cfg.Flags.Snapshot
	if snapName == "" && r.cfg.Flags.CreateSnapshot {
		if snapName, err = r.createDataSnapshot(ctx, ve); err != nil {
			log.Fatal("Error creating DataSnapshot: %v", err)
		}
	} else if snapName != "" {
		snap := &v1alpha1.DataSnapshot{}
		if err := r.k8s.Get(ctx, k8s.Key(platform.Namespace, snapName), snap); err != nil {
			log.Fatal("Error getting DataSnapshot: %v", err)
		}
	}

	updatedVE := origVE.DeepCopy()
	updatedVE.Spec.Release = &v1alpha1.Release{
		AppDeployment: v1alpha1.ReleaseAppDeployment{
			Name:    appDep.Name,
			Version: appDep.Spec.Version,
		},
		DataSnapshot: snapName,
	}
	if err := r.k8s.Merge(ctx, updatedVE, origVE); err != nil {
		log.Fatal("Error updating Release: %v", err)
	}

	r.waitForReady(platform, &appDep.Spec)

	return updatedVE
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

func (r *repo) createDataSnapshot(ctx context.Context, ve *v1alpha1.VirtualEnvironment) (string, error) {
	log.Verbose("checking for existing DataSnapshot of VirtualEnvironment '%s' with resourceVersion '%s'",
		ve.Name, ve.ResourceVersion)

	list := &v1alpha1.DataSnapshotList{}
	if err := r.k8s.List(ctx, list, client.MatchingLabels{
		api.LabelK8sSourceKind:    string(api.DataSourceKindVirtualEnvironment),
		api.LabelK8sSourceName:    ve.Name,
		api.LabelK8sSourceVersion: ve.ResourceVersion,
	}); err != nil {
		return "", err
	}
	dataSource := v1alpha1.DataSource{
		Kind:            api.DataSourceKindVirtualEnvironment,
		Name:            ve.Name,
		ResourceVersion: ve.ResourceVersion,
		DataChecksum:    ve.GetDataChecksum(),
	}

	for _, s := range list.Items {
		// Double check source is equal.
		if s.Spec.Source == dataSource {
			log.Verbose("found existing snapshot '%s'", s.Name)
			return s.Name, nil
		}
	}

	log.VerboseMarshal(ve, "creating DataSnapshot")
	dataSnap := &v1alpha1.DataSnapshot{
		TypeMeta: v1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       "DataSnapshot",
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: ve.Namespace,
			Name: fmt.Sprintf("%s-%s-%s",
				ve.Name, ve.ResourceVersion, time.Now().UTC().Format("20060102-150405")),
		},
		Spec: v1alpha1.DataSnapshotSpec{
			Source: dataSource,
		},
	}
	if err := r.k8s.Create(ctx, dataSnap); err != nil {
		return "", err
	}

	return dataSnap.Name, nil
}
