package repo

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/kubefox/api"
	"github.com/xigxog/kubefox/api/kubernetes/v1alpha1"
	"github.com/xigxog/kubefox/core"
	"github.com/xigxog/kubefox/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *repo) Release(appDepId string) *v1alpha1.Release {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	platform, err := r.k8s.GetPlatform(ctx)
	if err != nil {
		log.Fatal("Error getting Platform: %v", err)
	}

	appDep, err := r.findAppDep(ctx, platform, appDepId)
	if err != nil {
		log.Fatal("Error finding AppDeployment: %v", err)
	}

	envObj, err := r.k8s.SnapshotVirtualEnv(ctx, platform.Namespace, r.cfg.Flags.VirtEnv)
	if err != nil {
		log.Fatal("Error finding VirtualEnvironment: %v", err)
	}

	var envSnapshotName string
	if r.cfg.Flags.CreateVirtEnv {
		if name, err := r.createEnvSnapshot(ctx, envObj); err != nil {
			log.Fatal("Error creating VirtualEnvironmentSnapshot: %v", err)
		} else {
			envSnapshotName = name
		}

	} else if envObj.Data.Source.Kind == "VirtualEnvSnapshot" {
		envSnapshotName = envObj.Name
	}

	rel := &v1alpha1.Release{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       "Release",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      envObj.GetEnvName(),
			Namespace: platform.Namespace,
		},
		Spec: v1alpha1.ReleaseSpec{
			VirtualEnvSnapshot: envSnapshotName,
			AppDeployment: v1alpha1.ReleaseAppDeployment{
				Name:    appDep.Name,
				Version: appDep.Spec.Version,
			},
		},
	}

	if err := r.k8s.Upsert(ctx, rel); err != nil {
		log.Fatal("Error creating release: %v", err)
	}

	r.waitForReady(platform, &appDep.Spec)

	return rel
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

func (r *repo) createEnvSnapshot(ctx context.Context, envObj *v1alpha1.VirtualEnvSnapshot) (string, error) {
	if envObj.Data.Source.Kind == "VirtualEnvSnapshot" {
		return "", fmt.Errorf("cannot create snapshot of existing VirtualEnvSnapshot")
	}

	log.Verbose("checking for existing snapshot of '%s' with resource version '%s'",
		envObj.Data.Source.Name, envObj.Data.Source.ResourceVersion)

	list := &v1alpha1.VirtualEnvSnapshotList{}
	if err := r.k8s.List(ctx, list, client.MatchingLabels{
		api.LabelK8sVirtualEnv:            envObj.Data.Source.Name,
		api.LabelK8sSourceKind:            envObj.Data.Source.Kind,
		api.LabelK8sSourceResourceVersion: envObj.Data.Source.ResourceVersion,
	}); err != nil {
		return "", err
	}
	for _, snap := range list.Items {
		// Double check source is equal.
		if snap.Data.Source == envObj.Data.Source {
			log.Verbose("found existing snapshot '%s'", list.Items[0].Name)
			return snap.Name, nil
		}
	}

	log.VerboseMarshal(envObj, "creating VirtualEnvSnapshot")
	if err := r.k8s.Create(ctx, envObj); err != nil {
		return "", err
	}

	return envObj.Name, nil
}
