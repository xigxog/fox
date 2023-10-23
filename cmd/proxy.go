package cmd

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/proxy"
)

var proxyCmd = &cobra.Command{
	Use:    "proxy [local port]",
	Args:   cobra.ExactArgs(1),
	PreRun: setup,
	Run:    runProxy,
	Short:  "Port forward local port to broker's HTTP server adapter",
	Long: `
The proxy command will inspect the Kubernetes cluster and find an available
broker to proxy a local port to. This port can then be used to make HTTP
requests to the broker's HTTP server adapter. This is especially useful during
development and testing.

The optional flags 'env' and 'deployment' can be set which will automatically
inject the values as context to requests sent through the proxy. The context
can still be overridden manually by setting the header or query param on the 
original request.

Examples:
##### Port forward local port 8080 and wait if no brokers are available.
fox proxy 8080 --wait 5m

##### Port forward local port 8080 and inject 'my-env' and 'my-dep' context.
fox proxy 8080 --env my-env --deployment my-dep

	http://127.0.0.1:8080/hello                 # uses my-env and my-deployment
	http://127.0.0.1:8080/hello?kf-env=your-env # uses your-env and my-dep
	http://127.0.0.1:8080/hello?kf-dep=your-dep # uses my-env and your-dep
`,
}

func init() {
	proxyCmd.Flags().StringVarP(&cfg.Flags.Env, "env", "e", "", "environment to add to proxied requests")
	proxyCmd.Flags().StringVarP(&cfg.Flags.Deployment, "deployment", "d", "", "deployment to add to proxied requests")

	addCommonDeployFlags(proxyCmd)
	rootCmd.AddCommand(proxyCmd)
}

func runProxy(cmd *cobra.Command, args []string) {
	port, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatal("Error invalid local port '%s'.", args[0])
	}

	proxy.Start(port, cfg)
}
