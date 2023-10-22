package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/kubernetes"
	"github.com/xigxog/fox/internal/log"
)

var proxyCmd = &cobra.Command{
	Use:    "proxy [local port]",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    proxy,
	Short:  "Port forward local port to broker's HTTP server adapter",
	Long: `
The proxy command will inspect the Kubernetes cluster and find an available
broker to forward a local port to. This port can then be used to make HTTP
requests to the broker's HTTP server adapter. This is especially useful during
development and testing.

Examples:
# Port forward local port 8080 and wait if no brokers are available.
fox proxy 8080 --wait
`,
}

func init() {
	addCommonDeployFlags(proxyCmd)
	rootCmd.AddCommand(proxyCmd)
}

func proxy(cmd *cobra.Command, args []string) {
	port, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatal("Error invalid local port '%s'.", args[0])
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c := kubernetes.NewClient(cfg)

	p, err := c.GetPlatform(ctx)
	if err != nil {
		log.Fatal("Error getting platform :%v", err)
	}

	pfReq := &kubernetes.PortForwardRequest{
		Namespace: p.Namespace,
		Platform:  p.Name,
		LocalPort: int32(port),
	}
	pf, err := c.PortForward(ctx, pfReq)
	if errors.Is(err, kubernetes.ErrComponentNotRead) && cfg.Flags.WaitTime > 0 {
		log.Warn("No broker pod is available.")
		log.Info("Waiting for broker pod to become available...")

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Flags.WaitTime)
		defer cancel()

		err = c.WaitPodReady(ctx, p, "broker", "")
		if err == nil {
			pf, err = c.PortForward(ctx, pfReq)
		}
	}
	if err != nil {
		log.Fatal("Error starting proxy: %v", err)
	}

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	go func() {
		<-interruptCh
		pf.Stop()
	}()

	log.Info("The proxy is ready. You can now make HTTP requests on '127.0.0.1:%d'. If you are", port)
	log.Info("working on the quickstart try opening 'http://127.0.0.1:8080/hello' in your")
	log.Info("browser. If you get 'route not found' you probably haven't released the app yet.")
	log.Info("Try adding context to the request, 'http://localhost:30080/hello?kf-dep=my-deployment&kf-env=world'")
	log.Printf("HTTP proxy started on 127.0.0.1:%d\n", port)
	<-pf.Done()
}
