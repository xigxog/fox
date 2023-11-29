package kubernetes

import (
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"bytes"
	"net"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/xigxog/fox/internal/config"
	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/utils"
	"github.com/xigxog/kubefox/api"
	"github.com/xigxog/kubefox/api/kubernetes/v1alpha1"
	"github.com/xigxog/kubefox/k8s"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrComponentNotReady = fmt.Errorf("component not ready")
)

type Client struct {
	*k8s.Client

	cfg *config.Config
}

type PortForwardRequest struct {
	Namespace   string
	Platform    string
	HTTPSrvPod  string
	HTTPSrvPort int32
	LocalPort   int32
}

type PortForward struct {
	LocalPort int32

	pfer    *portforward.PortForwarder
	stopCh  chan struct{}
	readyCh chan struct{}
}

func NewClient(cfg *config.Config) *Client {
	cli, err := k8s.NewClient("fox")
	if err != nil {
		log.Fatal("Error creating Kubernetes client: %v", err)
	}

	return &Client{
		Client: cli,
		cfg:    cfg,
	}
}

func (c *Client) Create(ctx context.Context, obj client.Object) error {
	opts := []client.CreateOption{}
	if c.cfg.Flags.DryRun {
		opts = append(opts, client.DryRunAll)
	}
	return c.Client.Create(ctx, obj, opts...)
}

func (c *Client) Upsert(ctx context.Context, obj client.Object) error {
	return c.Client.Upsert(ctx, obj, c.cfg.Flags.DryRun)
}

func (c *Client) Apply(ctx context.Context, obj client.Object) error {
	opts := []client.PatchOption{}
	if c.cfg.Flags.DryRun {
		opts = append(opts, client.DryRunAll)
	}
	return c.Client.Apply(ctx, obj, opts...)
}

func (r *Client) ListPlatforms(ctx context.Context) ([]v1alpha1.Platform, error) {
	pList := &v1alpha1.PlatformList{}
	if err := r.Client.List(ctx, pList); err != nil {
		return nil, fmt.Errorf("unable to list KubeFox Platforms: %w", err)
	}

	return pList.Items, nil
}

func (c *Client) GetPlatform(ctx context.Context) (*v1alpha1.Platform, error) {
	nn := client.ObjectKey{
		Namespace: c.cfg.Flags.Namespace,
		Name:      c.cfg.Flags.Platform,
	}
	if nn.Name == "" {
		nn.Namespace = c.cfg.KubeFox.Namespace
		nn.Name = c.cfg.KubeFox.Platform

	}

	platform := &v1alpha1.Platform{}
	if nn.Name == "" {
		platform = c.pickPlatform()

	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		if err := c.Get(ctx, nn, platform); err != nil {
			if apierrors.IsNotFound(err) {
				platform = c.pickPlatform()
			} else {
				log.Fatal("Unable to get KubeFox Platform: %v", err)
			}
		}
	}

	return platform, nil
}

func (c *Client) pickPlatform() *v1alpha1.Platform {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pList, err := c.ListPlatforms(ctx)
	if err != nil {
		log.Fatal("%v", err)
	}

	switch len(pList) {
	case 0:
		if !c.cfg.Flags.Info {
			context := c.Client.KubeConfig.CurrentContext
			cluster := c.Client.KubeConfig.Contexts[context].Cluster
			log.Warn("No KubeFox Platforms found on the current cluster '%s'.", cluster)
		}
		log.Info("You need to have a KubeFox Platform instance running to deploy your components.")
		log.Info("Don't worry, 🦊 Fox can create one for you.")
		if utils.YesNoPrompt("Would you like to create a KubeFox Platform?", true) {
			return c.createPlatform()
		} else {
			log.Fatal("Error you must create a KubeFox Platform before deploying components.")
		}
	case 1:
		return &pList[0]
	}

	for i, p := range pList {
		log.Printf("%d. %s/%s\n", i+1, p.Namespace, p.Name)
	}

	var input string
	log.Printf("Select the KubeFox Platform to use: ")
	fmt.Scanln(&input)
	i, err := strconv.Atoi(input)
	if err != nil {
		return c.pickPlatform()
	}
	i = i - 1
	if i < 0 || i >= len(pList) {
		return c.pickPlatform()
	}

	p := &pList[i]
	if len(pList) > 1 {
		if utils.YesNoPrompt("Remember selected KubeFox Platform?", true) {
			c.cfg.KubeFox.Namespace = p.Namespace
			c.cfg.KubeFox.Platform = p.Name
			c.cfg.Write()
		}
	}
	log.InfoNewline()

	return p
}

func (c *Client) createPlatform() *v1alpha1.Platform {
	name := utils.NamePrompt("KubeFox Platform", "", true)
	namespace := utils.InputPrompt("Enter the Kubernetes namespace of the KubeFox Platform",
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
	if err := c.Apply(ctx, ns); err != nil {
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
	if err := c.Apply(ctx, p); err != nil {
		log.Fatal("%v", err)
	}

	return p
}

func (c *Client) WaitPlatformReady(ctx context.Context, p *v1alpha1.Platform, spec *v1alpha1.AppDeploymentSpec) {
	log.Info("Waiting for KubeFox Platform '%s' to be ready...", p.Name)
	if err := c.WaitPodReady(ctx, p, "nats", ""); err != nil {
		log.Fatal("Error while waiting: %v", err)
	}
	if err := c.WaitPodReady(ctx, p, "broker", ""); err != nil {
		log.Fatal("Error while waiting: %v", err)
	}

	for n, comp := range spec.Components {
		log.Info("Waiting for component '%s' to be ready...", n)
		if err := c.WaitPodReady(ctx, p, n, comp.Commit); err != nil {
			log.Fatal("Error while waiting: %v", err)
		}
	}
	log.InfoNewline()
}

func (c *Client) WaitPodReady(ctx context.Context, p *v1alpha1.Platform, comp, commit string) error {
	log.Verbose("Waiting for component '%s' with commit '%s' to be ready...", comp, commit)

	hasLabels := client.MatchingLabels{
		api.LabelK8sComponent: comp,
		api.LabelK8sPlatform:  p.Name,
	}
	if commit != "" {
		hasLabels[api.LabelK8sComponentCommit] = commit
	}

	l := &corev1.PodList{}
	if err := c.List(ctx, l, client.InNamespace(p.Namespace), hasLabels); err != nil {
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
		return c.WaitPodReady(ctx, p, comp, commit)
	}
	log.Verbose("Component '%s' with commit '%s' is ready.", comp, commit)

	return nil
}

func (c *Client) PortForward(ctx context.Context, req *PortForwardRequest) (*PortForward, error) {
	if req.HTTPSrvPod == "" {
		podList := &corev1.PodList{}
		err := c.List(ctx, podList,
			client.InNamespace(req.Namespace),
			client.MatchingLabels{
				api.LabelK8sPlatform:  req.Platform,
				api.LabelK8sComponent: "httpsrv",
			},
		)
		if err != nil {
			return nil, err
		}
		if len(podList.Items) == 0 {
			return nil, fmt.Errorf("%w: no httpsrv pods found", ErrComponentNotReady)
		}

		var name string
		for _, p := range podList.Items {
			var ready bool
			for _, c := range p.Status.ContainerStatuses {
				if c.Ready {
					ready = true
					break
				}
			}
			log.Verbose("pod: %s, phase: %s, ready: %t", p.Name, p.Status.Phase, ready)
			if !ready {
				continue
			}
			name = p.Name
			break
		}
		if name == "" {
			return nil, fmt.Errorf("%w: no available httpsrv pod", ErrComponentNotReady)
		}
		req.HTTPSrvPod = podList.Items[0].Name
	}
	if req.HTTPSrvPort == 0 {
		pod := &corev1.Pod{}
		err := c.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.HTTPSrvPod}, pod)
		if err != nil {
			return nil, err
		}
		for _, c := range pod.Spec.Containers {
			if c.Name != "httpsrv" {
				continue
			}
			for _, p := range c.Ports {
				if p.Name != "http" {
					continue
				}
				req.HTTPSrvPort = p.ContainerPort
				break
			}
		}
		if req.HTTPSrvPort == 0 {
			req.HTTPSrvPort = 8080
		}
	}
	if req.LocalPort == 0 {
		// Find available local port.
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			return nil, err
		}
		req.LocalPort = int32(l.Addr().(*net.TCPAddr).Port)
		if err := l.Close(); err != nil {
			return nil, err
		}
	}

	pf := &PortForward{
		LocalPort: req.LocalPort,
		stopCh:    make(chan struct{}, 1),
		readyCh:   make(chan struct{}),
	}

	scheme := "https"
	host := c.Client.RestConfig.Host
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", req.Namespace, req.HTTPSrvPod)
	if u, err := url.Parse(host); err == nil { // success
		scheme = u.Scheme
		host = u.Host
		path = fmt.Sprintf("%s%s", u.Path, path)
	}

	transport, upgrader, err := spdy.RoundTripperFor(c.Client.RestConfig)
	if err != nil {
		return nil, err
	}

	dialer := spdy.NewDialer(
		upgrader,
		&http.Client{Transport: transport},
		http.MethodPost,
		&url.URL{
			Scheme: scheme,
			Path:   path,
			Host:   host,
		})

	var buf bytes.Buffer
	pfer, err := portforward.New(
		dialer,
		[]string{fmt.Sprintf("%d:%d", req.LocalPort, req.HTTPSrvPort)},
		pf.stopCh, pf.readyCh, &buf, &buf)
	if err != nil {
		return nil, err
	}
	pf.pfer = pfer

	go func() {
		if err := pfer.ForwardPorts(); err != nil {
			log.Fatal("Error with port forward: %v", err)
		}
	}()

	// Wait for port forward to be ready.
	<-pf.readyCh
	log.Verbose("Port forward ready; pod: '%s', podPort: '%d', localPort: '%d'.",
		req.HTTPSrvPod, req.HTTPSrvPort, req.LocalPort)

	return pf, nil
}

func (pf *PortForward) Stop() {
	pf.pfer.Close()
	close(pf.stopCh)
}

func (pf *PortForward) Ready() <-chan struct{} {
	return pf.readyCh
}

func (pf *PortForward) Done() <-chan struct{} {
	return pf.stopCh
}
