package kubernetes

import (
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"context"
	"fmt"

	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/api/kubernetes/v1alpha1"
)

const (
	FieldOwner client.FieldOwner = "fox"
)

type Client struct {
	client.Client
}

func NewClient() *Client {
	v1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal("Error reading Kubernetes config file: %v", err)
	}
	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		log.Fatal("Error setting up Kubernetes client: %v", err)
	}

	return &Client{Client: c}
}

func (c *Client) Apply(ctx context.Context, obj client.Object) error {
	return c.Patch(ctx, obj, client.Apply, FieldOwner, client.ForceOwnership)
}

func (c *Client) Merge(ctx context.Context, obj client.Object) error {
	return c.Patch(ctx, obj, client.Merge, FieldOwner)
}

func (r *Client) ListPlatforms(ctx context.Context) ([]v1alpha1.Platform, error) {
	pList := &v1alpha1.PlatformList{}
	if err := r.List(ctx, pList); err != nil {
		return nil, fmt.Errorf("unable to fetch platforms: %w", err)
	}

	return pList.Items, nil
}
