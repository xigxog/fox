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
	"github.com/xigxog/kubefox/libs/api/kubernetes/v1alpha1"
	"github.com/xigxog/kubefox/libs/core/kubefox"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	ErrComponentNotRead = fmt.Errorf("component not ready")
)

const (
	FieldOwner client.FieldOwner = "fox"
)

type Client struct {
	client.Client

	KubeConfig *clientcmdapi.Config
	RestConfig *rest.Config

	cfg *config.Config
}

type PortForwardRequest struct {
	Namespace  string
	Platform   string
	BrokerPod  string
	BrokerPort int32
	LocalPort  int32
}

type PortForward struct {
	LocalPort int32

	pfer    *portforward.PortForwarder
	stopCh  chan struct{}
	readyCh chan struct{}
}

func NewClient(cfg *config.Config) *Client {
	v1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)

	l := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig, err := l.Load()
	if err != nil {
		log.Fatal("Error reading Kubernetes config file: %v", err)
	}

	kCfg, err := kconfig.GetConfig()
	if err != nil {
		log.Fatal("Error reading Kubernetes config file: %v", err)
	}
	c, err := client.New(kCfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		log.Fatal("Error setting up Kubernetes client: %v", err)
	}

	return &Client{
		Client:     c,
		KubeConfig: kubeConfig,
		RestConfig: kCfg,
		cfg:        cfg,
	}
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

func (c *Client) GetPlatform(ctx context.Context) (*v1alpha1.Platform, error) {
	nn := types.NamespacedName{
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
				log.Fatal("Unable to get KubeFox platform: %v", err)
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
			context := c.KubeConfig.CurrentContext
			cluster := c.KubeConfig.Contexts[context].Cluster
			log.Warn("No KubeFox platforms found on the current cluster '%s'.", cluster)
		}
		log.Info("You need to have a KubeFox platform instance running to deploy your components.")
		log.Info("Don't worry, 🦊 Fox can create one for you.")
		if utils.YesNoPrompt("Would you like to create a KubeFox platform?", true) {
			return c.createPlatform()
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
		return c.pickPlatform()
	}
	i = i - 1
	if i < 0 || i >= len(pList) {
		return c.pickPlatform()
	}

	p := &pList[i]
	if len(pList) > 1 {
		if utils.YesNoPrompt("Remember selected KubeFox platform?", true) {
			c.cfg.KubeFox.Namespace = p.Namespace
			c.cfg.KubeFox.Platform = p.Name
			c.cfg.Write()
		}
	}
	log.InfoNewline()

	return p
}

func (c *Client) createPlatform() *v1alpha1.Platform {
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

func (c *Client) WaitPlatformReady(ctx context.Context, p *v1alpha1.Platform, spec *v1alpha1.DeploymentSpec) {
	log.Info("Waiting for KubeFox platform '%s' to be ready...", p.Name)
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
		kubefox.LabelK8sComponent: comp,
		kubefox.LabelK8sPlatform:  p.Name,
	}
	if commit != "" {
		hasLabels[kubefox.LabelK8sComponentCommit] = commit
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
	if req.BrokerPod == "" {
		podList := &corev1.PodList{}
		err := c.List(ctx, podList,
			client.InNamespace(req.Namespace),
			client.MatchingLabels{
				kubefox.LabelK8sPlatform:  req.Platform,
				kubefox.LabelK8sComponent: "broker",
			},
		)
		if err != nil {
			return nil, err
		}
		if len(podList.Items) == 0 {
			return nil, fmt.Errorf("%w: no broker pods found", ErrComponentNotRead)
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
			return nil, fmt.Errorf("%w: no available broker pod", ErrComponentNotRead)
		}
		req.BrokerPod = podList.Items[0].Name
	}
	if req.BrokerPort == 0 {
		pod := &corev1.Pod{}
		err := c.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.BrokerPod}, pod)
		if err != nil {
			return nil, err
		}
		for _, c := range pod.Spec.Containers {
			if c.Name != "broker" {
				continue
			}
			for _, p := range c.Ports {
				if p.Name != "http" {
					continue
				}
				req.BrokerPort = p.ContainerPort
				break
			}
		}
		if req.BrokerPort == 0 {
			req.BrokerPort = 8080
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
	host := c.RestConfig.Host
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", req.Namespace, req.BrokerPod)
	if u, err := url.Parse(host); err == nil { // success
		scheme = u.Scheme
		host = u.Host
		path = fmt.Sprintf("%s%s", u.Path, path)
	}

	transport, upgrader, err := spdy.RoundTripperFor(c.RestConfig)
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
		[]string{fmt.Sprintf("%d:%d", req.LocalPort, req.BrokerPort)},
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
		req.BrokerPod, req.BrokerPort, req.LocalPort)

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
