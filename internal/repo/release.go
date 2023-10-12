package repo

import (
	"context"
	"time"

	"github.com/xigxog/kubefox-cli/internal/config"
	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/api/kubernetes/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *repo) Release(name string) *v1alpha1.Release {
	p, spec := r.buildDepSpec()

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
				Name:            name,
				UID:             types.UID(config.Flags.EnvUID),
				ResourceVersion: config.Flags.EnvVersion,
			},
		},
	}

	if err := r.k8s.Apply(ctx, rel); err != nil {
		log.Fatal("%v", err)
	}

	return rel
}
