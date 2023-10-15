package repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox-cli/internal/utils"
	"github.com/xigxog/kubefox/libs/api/kubernetes/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *repo) Deploy(name string) *v1alpha1.Deployment {
	p, spec := r.buildDepSpec()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	r.applyIPS(ctx, p, spec)

	d := &v1alpha1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.Namespace,
			Name:      name,
		},
		Spec: *spec,
	}
	log.VerboseMarshal(d, "deployment:")

	if err := r.k8s.Apply(ctx, d); err != nil {
		log.Fatal("%v", err)
	}

	return d
}

func (r *repo) applyIPS(ctx context.Context, p *v1alpha1.Platform, spec *v1alpha1.DeploymentSpec) {
	if r.cfg.ContainerRegistry.Token != "" {
		cr := r.cfg.ContainerRegistry
		name := fmt.Sprintf("%s-image-pull-secret", spec.App.Name)
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
		spec.App.ImagePullSecret = name
	}
}

func (r *repo) buildDepSpec() (*v1alpha1.Platform, *v1alpha1.DeploymentSpec) {
	nn := types.NamespacedName{
		Namespace: r.cfg.Flags.Namespace,
		Name:      r.cfg.Flags.Platform,
	}
	if nn.Name == "" {
		nn.Namespace = r.cfg.KubeFox.Namespace
		nn.Name = r.cfg.KubeFox.Platform

	}

	platform := &v1alpha1.Platform{}
	if nn.Name == "" {
		platform = r.pickPlatform()

	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		if err := r.k8s.Get(ctx, nn, platform); err != nil {
			if apierrors.IsNotFound(err) {
				platform = r.pickPlatform()
			} else {
				log.Fatal("Unable to get Platform: %v", err)
			}
		}
	}

	return platform, r.getDepSpec()
}

func (r *repo) getDepSpec() *v1alpha1.DeploymentSpec {
	compsDirPath := filepath.Join(r.cfg.Flags.RepoPath, ComponentsDirName)
	compsDir, err := os.ReadDir(compsDirPath)
	if err != nil {
		log.Fatal("Error listing components dir '%s': %v", compsDirPath, err)
	}

	depSpec := &v1alpha1.DeploymentSpec{}
	depSpec.App.Name = r.app.Name
	depSpec.App.Title = r.app.Title
	depSpec.App.Description = r.app.Description
	depSpec.App.GitRepo = r.GetRepoURL()
	depSpec.App.GitRef = r.GetRefName()
	depSpec.App.Commit = r.GetCommit("")
	if r.app.ContainerRegistry != "" {
		depSpec.App.ContainerRegistry = r.app.ContainerRegistry
	} else {
		depSpec.App.ContainerRegistry = fmt.Sprintf("%s/%s", r.cfg.ContainerRegistry.Address, r.app.Name)
	}

	depSpec.Components = map[string]*v1alpha1.Component{}
	for _, compDir := range compsDir {
		if !compDir.IsDir() {
			continue
		}

		compName := utils.Clean(compDir.Name())
		depSpec.Components[compName] = &v1alpha1.Component{
			Commit: r.GetCompCommit(compDir.Name()),
		}
	}

	return depSpec
}

func (r *repo) pickPlatform() *v1alpha1.Platform {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pList, err := r.k8s.ListPlatforms(ctx)
	if err != nil {
		log.Fatal("%v", err)
	}

	switch len(pList) {
	case 0:
		log.Error("No Platforms found. Ensure you have the correct context active (`kubectl config get-contexts`).")
		log.Error("If so, Fox can create a Platform for you.")
		if utils.YesNoPrompt("Would you like to create a Platform?", true) {
			return r.createPlatform()
		} else {
			log.Fatal("Error you must create a Platform before deploying Components.")
		}
	case 1:
		return &pList[0]
	}

	for i, p := range pList {
		log.Info("%d. %s/%s", i+1, p.Namespace, p.Name)
	}

	var input string
	log.Printf("Select the KubeFox Platform to use: ")
	fmt.Scanln(&input)
	i, err := strconv.Atoi(input)
	if err != nil {
		return r.pickPlatform()
	}
	i = i - 1
	if i < 0 || i >= len(pList) {
		return r.pickPlatform()
	}

	p := &pList[i]
	if len(pList) > 1 {
		if utils.YesNoPrompt("Remember selected Platform?", true) {
			r.cfg.KubeFox.Namespace = p.Namespace
			r.cfg.KubeFox.Platform = p.Name
			r.cfg.Write()
		}
	}

	return p
}

func (r *repo) createPlatform() *v1alpha1.Platform {
	name := utils.NamePrompt("Platform", "", true)
	namespace := utils.InputPrompt("Enter the Kubernetes namespace of the Platform", "", true)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.Identifier(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	if err := r.k8s.Apply(ctx, ns); err != nil {
		log.Fatal("%v", err)
	}

	p := &v1alpha1.Platform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       "Platform",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := r.k8s.Apply(ctx, p); err != nil {
		log.Fatal("%v", err)
	}

	return p
}
