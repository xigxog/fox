package repo

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/utils"
	"github.com/xigxog/kubefox/libs/api/kubernetes/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *repo) Deploy(name string, skipImageCheck bool) *v1alpha1.Deployment {
	p, spec := r.prepareDeployment(skipImageCheck)

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

func (r *repo) Publish(deployName string) *v1alpha1.Deployment {
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
func (r *repo) prepareDeployment(skipImageCheck bool) (*v1alpha1.Platform, *v1alpha1.DeploymentSpec) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	spec := r.getDepSpec()
	platform, err := r.k8s.GetPlatform(ctx)
	if err != nil {
		log.Fatal("Error getting platform :%v", err)
	}

	if !skipImageCheck {
		allFound := true
		for n, c := range spec.Components {
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
			if utils.YesNoPrompt("Missing component images, would you like to publish them?", true) {
				log.InfoNewline()
				r.Publish("")
			} else {
				log.Fatal("There are one or more missing component images.")
			}
		}
	}

	return platform, spec
}

func (r *repo) getDepSpec() *v1alpha1.DeploymentSpec {
	compsDir, err := os.ReadDir(r.ComponentsDir())
	if err != nil {
		log.Fatal("Error listing components dir '%s': %v", r.ComponentsDir(), err)
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

func (r *repo) waitForReady(p *v1alpha1.Platform, spec *v1alpha1.DeploymentSpec) {
	if r.cfg.Flags.WaitTime <= 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.cfg.Flags.WaitTime)
	defer cancel()

	r.k8s.WaitPlatformReady(ctx, p, spec)
}
