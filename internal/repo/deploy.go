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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/xigxog/fox/internal/log"
	foxutils "github.com/xigxog/fox/internal/utils"
	"github.com/xigxog/kubefox/api"
	"github.com/xigxog/kubefox/api/kubernetes/v1alpha1"
	"github.com/xigxog/kubefox/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *repo) Deploy(name string, skipImageCheck bool) *v1alpha1.AppDeployment {
	if r.cfg.Flags.CreateTag && !strings.HasSuffix(r.GetTagRef(), r.cfg.Flags.Version) {
		r.CreateTag(r.cfg.Flags.Version)
	}

	p, appDep := r.prepareDeployment(skipImageCheck)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	r.applyIPS(ctx, p, &appDep.Spec)

	appDep.ObjectMeta = metav1.ObjectMeta{
		Namespace: p.Namespace,
		Name:      name,
	}

	log.VerboseMarshal(appDep, "AppDeployment:")

	if err := r.k8s.Merge(ctx, appDep, nil); err != nil {
		log.Fatal("%v", err)
	}

	r.waitForReady(p, &appDep.Spec)

	return appDep
}

func (r *repo) Publish(deployName string) *v1alpha1.AppDeployment {
	compsDir, err := os.ReadDir(r.ComponentsDir())
	if err != nil {
		log.Fatal("Error listing components dir '%s': %v", r.ComponentsDir(), err)
	}

	for _, compDir := range compsDir {
		if !compDir.IsDir() {
			continue
		}
		r.Build(compDir.Name())
		log.InfoNewline()
	}

	if !r.cfg.Flags.SkipDeploy && deployName != "" {
		return r.Deploy(deployName, true)
	}

	return nil
}

func (r *repo) applyIPS(ctx context.Context, p *v1alpha1.Platform, spec *v1alpha1.AppDeploymentSpec) {
	if r.cfg.ContainerRegistry.Token != "" {
		cr := r.cfg.ContainerRegistry
		name := fmt.Sprintf("%s-image-pull-secret", spec.AppName)
		dockerCfg := fmt.Sprintf(`{"auths":{"%s":{"username":"kubefox","password":"%s"}}}`, cr.Address, cr.Token)

		s := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.Identifier(),
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: p.Namespace,
				Name:      name,
			},
			Type: "kubernetes.io/dockerconfigjson",

			StringData: map[string]string{
				".dockerconfigjson": dockerCfg,
			},
		}

		if err := r.k8s.Apply(ctx, s); err != nil {
			log.Fatal("%v", err)
		}
		spec.ImagePullSecretName = name
	}
}

// prepareDeployment pulls the Platform, generates the AppDeploymentSpec and
// ensures all images exist. If there are any issues it will prompt the user to
// correct them.
func (r *repo) prepareDeployment(skipImageCheck bool) (*v1alpha1.Platform, *v1alpha1.AppDeployment) {
	appDep := r.buildAppDep()
	platform := r.k8s.GetPlatform()

	if !skipImageCheck {
		allFound := true
		for n, c := range appDep.Spec.Components {
			img := r.GetCompImage(n, c.Commit)
			if found, _ := r.DoesImageExists(img, false); found {
				log.Info("Component image '%s' exists.", img)
				if r.cfg.IsRegistryLocal() {
					r.KindLoad(img)
				}
			} else {
				log.Warn("Component image '%s' does not exist.", img)
				allFound = false
			}
			log.InfoNewline()
		}

		if !allFound {
			log.Info("There are one or more missing component images. 🦊 Fox will need to build and")
			log.Info("push them to the container registry before continuing with the operation.")
			if foxutils.YesNoPrompt("Missing component images, would you like to publish them?", true) {
				log.InfoNewline()
				r.Publish("")
			} else {
				log.Fatal("There are one or more missing component images.")
			}
		}
	}

	for compName, comp := range appDep.Spec.Components {
		if err := r.extractCompDef(compName, comp); err != nil {
			log.Fatal("Error getting component '%s' definition: %v", compName, err)
		}
	}

	return platform, appDep
}

func (r *repo) buildAppDep() *v1alpha1.AppDeployment {
	compsDir, err := os.ReadDir(r.ComponentsDir())
	if err != nil {
		log.Fatal("Error listing components dir '%s': %v", r.ComponentsDir(), err)
	}
	commit := r.GetCommit("")
	reg := r.app.ContainerRegistry
	if reg == "" {
		reg = r.cfg.ContainerRegistry.Address
	}

	appDep := &v1alpha1.AppDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       "AppDeployment",
		},
		Spec: v1alpha1.AppDeploymentSpec{
			AppName:           r.app.Name,
			Commit:            commit.Hash.String(),
			CommitTime:        metav1.NewTime(commit.Committer.When),
			Version:           r.cfg.Flags.Version,
			RepoURL:           r.GetRepoURL(),
			Branch:            r.GetHeadRef(),
			Tag:               r.GetTagRef(),
			ContainerRegistry: reg,
			Components:        map[string]*api.ComponentDefinition{},
		},
		Details: v1alpha1.AppDeploymentDetails{
			Details: api.Details{
				Title:       r.app.Title,
				Description: r.app.Description,
			},
		},
	}

	for _, compDir := range compsDir {
		if !compDir.IsDir() {
			continue
		}
		compName := utils.CleanName(compDir.Name())
		appDep.Spec.Components[compName] = &api.ComponentDefinition{
			Commit: r.GetCompCommit(compDir.Name()).Hash.String(),
		}
	}

	return appDep
}

func (r *repo) extractCompDef(compName string, comp *api.ComponentDefinition) error {
	commit := comp.Commit
	img := r.GetCompImage(compName, comp.Commit)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	resp, err := r.docker.ContainerCreate(ctx, &container.Config{
		Image: img,
		Cmd:   []string{"-export"},
		Tty:   true,
	}, nil, nil, nil, "")
	if err != nil {
		return err
	}

	defer func() {
		if err := r.docker.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{}); err != nil {
			log.Error("Error removing component container: %v", err)
		}
	}()

	if err := r.docker.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	statusCh, errCh := r.docker.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-statusCh:
	}

	out, err := r.docker.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return err
	}
	b, err := io.ReadAll(out)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, comp); err != nil {
		return err
	}
	comp.Commit = commit

	return nil
}

func (r *repo) waitForReady(p *v1alpha1.Platform, spec *v1alpha1.AppDeploymentSpec) {
	if r.cfg.Flags.WaitTime <= 0 || r.cfg.Flags.DryRun {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.cfg.Flags.WaitTime)
	defer cancel()

	r.k8s.WaitPlatformReady(ctx, p, spec)
	log.InfoNewline()
}
