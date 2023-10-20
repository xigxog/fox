package repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/utils"
	"github.com/xigxog/kubefox/libs/api/kubernetes/v1alpha1"
	"github.com/xigxog/kubefox/libs/core/kubefox"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *repo) Deploy(name string) *v1alpha1.Deployment {
	p, spec := r.prepareDeployment()

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
	if err := r.k8s.Apply(ctx, d); err != nil {
		log.Fatal("%v", err)
	}

	r.waitForReady(p, spec)

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

// prepareDeployment pulls the Platform, generates the DeploymentSpec and
// ensures all images exist. If there are any issues it will prompt the user to
// correct them.
func (r *repo) prepareDeployment() (*v1alpha1.Platform, *v1alpha1.DeploymentSpec) {
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
				log.Fatal("Unable to get KubeFox platform: %v", err)
			}
		}
	}

	p, spec := platform, r.getDepSpec()

	allFound := true
	for n, c := range spec.Components {
		img := r.GetCompImage(n, c.Commit)
		if found, _ := r.ensureImageExists(img, false); found {
			log.Info("Component image '%s' exists.", img)
		} else {
			log.Warn("Component image '%s' does not exist.", img)
			allFound = false
		}
	}
	log.InfoNewline()

	if !allFound {
		log.Info("There are one or more missing component images. 🦊 Fox will need to build and")
		log.Info("push them to the container registry before continuing with the operation.")
		if utils.YesNoPrompt("Missing component images, would you like to publish them?", true) {
			log.InfoNewline()
			r.Publish("")
		} else {
			log.Fatal("There are one or more missing component images.")
		}
	}

	return p, spec
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
		if !r.cfg.Flags.Info {
			context := r.k8s.KubeConfig.CurrentContext
			cluster := r.k8s.KubeConfig.Contexts[context].Cluster
			log.Warn("No KubeFox platforms found on the current cluster '%s'.", cluster)
		}
		log.Info("You need to have a KubeFox platform instance running to deploy your components.")
		log.Info("Don't worry, 🦊 Fox can create one for you.")
		if utils.YesNoPrompt("Would you like to create a KubeFox platform?", true) {
			return r.createPlatform()
		} else {
			log.Fatal("Error you must create a KubeFox platform before deploying components.")
		}
	case 1:
		return &pList[0]
	}

	for i, p := range pList {
		log.Info("%d. %s/%s", i+1, p.Namespace, p.Name)
	}

	var input string
	log.Printf("Select the KubeFox platform to use: ")
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
		if utils.YesNoPrompt("Remember selected KubeFox platform?", true) {
			r.cfg.KubeFox.Namespace = p.Namespace
			r.cfg.KubeFox.Platform = p.Name
			r.cfg.Write()
		}
	}
	log.InfoNewline()

	return p
}

func (r *repo) createPlatform() *v1alpha1.Platform {
	name := utils.NamePrompt("KubeFox platform", "", true)
	namespace := utils.InputPrompt("Enter the Kubernetes namespace of the KubeFox platform",
		fmt.Sprintf("kubefox-%s", name), true)
	log.InfoNewline()

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

func (r *repo) waitForReady(p *v1alpha1.Platform, spec *v1alpha1.DeploymentSpec) {
	if r.cfg.Flags.WaitTime <= 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.cfg.Flags.WaitTime)
	defer cancel()

	log.Info("Waiting for KubeFox platform '%s' to be ready.", p.Name)
	if err := r.checkAllPodsRdy(ctx, p, "nats", ""); err != nil {
		log.Fatal("Error while waiting: %v", err)
	}
	if err := r.checkAllPodsRdy(ctx, p, "broker", ""); err != nil {
		log.Fatal("Error while waiting: %v", err)
	}

	for n, c := range spec.Components {
		log.Info("Waiting for component '%s' to be ready.", n)
		if err := r.checkAllPodsRdy(ctx, p, n, c.Commit); err != nil {
			log.Fatal("Error while waiting: %v", err)
		}
	}
	log.InfoNewline()
}

func (r *repo) checkAllPodsRdy(ctx context.Context, p *v1alpha1.Platform, comp, commit string) error {
	log.Verbose("Waiting for component '%s' with commit '%s' to be ready.", comp, commit)

	hasLabels := client.MatchingLabels{
		kubefox.LabelK8sComponent: comp,
		kubefox.LabelK8sPlatform:  p.Name,
	}
	if commit != "" {
		hasLabels[kubefox.LabelK8sComponentCommit] = commit
	}

	l := &corev1.PodList{}
	if err := r.k8s.List(ctx, l, client.InNamespace(p.Namespace), hasLabels); err != nil {
		return fmt.Errorf("unable to list pods: %w", err)
	}

	ready := len(l.Items) > 0
	for _, p := range l.Items {
		for _, c := range p.Status.ContainerStatuses {
			if !c.Ready {
				ready = false
				break
			}
		}
	}
	if !ready {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		time.Sleep(time.Second * 3)
		return r.checkAllPodsRdy(ctx, p, comp, commit)
	}
	log.Verbose("Component '%s' with commit '%s' is ready.", comp, commit)

	return nil
}
