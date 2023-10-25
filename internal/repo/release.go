package repo

import (
	"context"
	"time"

	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/kubefox/libs/api/kubernetes/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *repo) Release(name string) *v1alpha1.Release {
	p, spec := r.prepareDeployment(false)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	r.applyIPS(ctx, p, spec)

	rel := &v1alpha1.Release{
		TypeMeta: v1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       "Release",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: p.Namespace,
		},
		Spec: v1alpha1.ReleaseSpec{
			Deployment: *spec,
			Environment: v1alpha1.ReleaseEnv{
				Name:            r.cfg.Flags.Env,
				UID:             types.UID(r.cfg.Flags.EnvUID),
				ResourceVersion: r.cfg.Flags.EnvVersion,
			},
		},
	}

	// Hang on to typeMeta as it is erased by create.
	t := rel.TypeMeta
	err := r.k8s.Create(ctx, rel)
	if errors.IsAlreadyExists(err) {
		exRel := *rel
		if err := r.k8s.Get(ctx, types.NamespacedName{Namespace: rel.Namespace, Name: rel.Name}, &exRel); err != nil {
			log.Fatal("%v", err)
		}
		rel.ResourceVersion = exRel.ResourceVersion
		err = r.k8s.Update(ctx, rel)
	}
	if err != nil {
		log.Fatal("%v", err)
	}
	// Restore typeMeta.
	rel.TypeMeta = t

	r.waitForReady(p, spec)

	return rel
}
