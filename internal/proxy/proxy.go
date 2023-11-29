package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/xigxog/fox/internal/config"
	"github.com/xigxog/fox/internal/kubernetes"
	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/kubefox/api"
	kubefox "github.com/xigxog/kubefox/core"
)

type ProxyServer struct {
	cfg  *config.Config
	addr string

	httpSrv *http.Server
	client  http.Client
	pf      *kubernetes.PortForward
}

func Start(port int, cfg *config.Config) {
	log.Verbose("Starting HTTP proxy server...")

	srv := &ProxyServer{
		cfg: cfg,
		client: http.Client{
			Timeout: time.Minute,
		},
	}
	defer srv.Shutdown()

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	go func() {
		<-interruptCh
		srv.Shutdown()
	}()

	srv.pf = srv.startPortForward(cfg)

	srv.httpSrv = &http.Server{
		Handler: srv,
	}

	srv.addr = fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", srv.addr)
	if err != nil {
		log.Fatal("Error starting HTTP proxy: %v", err)
	}

	go func() {
		err := srv.httpSrv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Error running HTTP proxy server: %v", err)
		}
	}()

	log.Info("The proxy is ready. You can now make HTTP requests on 'http://%s'.", srv.addr)
	log.Info("If you are working on the quickstart try opening 'http://%s/hello'", srv.addr)
	log.Info("in your browser.")
	log.Printf("HTTP proxy started on http://%s\n", srv.addr)

	<-srv.pf.Done()
}

func (srv *ProxyServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h := fmt.Sprintf("127.0.0.1:%d", srv.pf.LocalPort)
	req.Host = h
	req.URL.Host = h
	req.URL.Scheme = "http"
	req.RequestURI = ""

	env := kubefox.GetParamOrHeader(req, api.HeaderEnv, api.HeaderAbbrvEnv, api.HeaderShortEnv)
	if env == "" && srv.cfg.Flags.VirtEnv != "" {
		req.Header.Set(api.HeaderEnv, srv.cfg.Flags.VirtEnv)
	}
	dep := kubefox.GetParamOrHeader(req, api.HeaderDep, api.HeaderAbbrvDep, api.HeaderShortDep)
	if dep == "" && srv.cfg.Flags.AppDeployment != "" {
		req.Header.Set(api.HeaderDep, srv.cfg.Flags.AppDeployment)
	}

	reqData, _ := httputil.DumpRequest(req, false)
	log.Verbose("Proxying request:\n%s", strings.TrimSpace(string(reqData)))

	resp, err := srv.client.Do(req)
	if err != nil {
		log.Error("Error proxying request: %v", err)
		return
	}
	defer resp.Body.Close()

	respData, _ := httputil.DumpResponse(resp, false)
	log.Verbose("Got response:\n%s", strings.TrimSpace(string(respData)))

	for k, v := range resp.Header {
		rw.Header()[k] = v
	}
	rw.WriteHeader(resp.StatusCode)

	_, err = io.Copy(rw, resp.Body)
	if err != nil {
		log.Error("Error writing response: %v", err)
		rw.Write([]byte("error"))
		return
	}
}

func (srv *ProxyServer) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if srv.httpSrv != nil {
		if err := srv.httpSrv.Shutdown(ctx); err != nil {
			log.Error("Error shutting down HTTP proxy server: %v", err)
		}
		srv.httpSrv = nil
		srv.pf.Stop()
	}

}

func (srv *ProxyServer) startPortForward(cfg *config.Config) *kubernetes.PortForward {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c := kubernetes.NewClient(cfg)
	p := c.GetPlatform()

	pfReq := &kubernetes.PortForwardRequest{
		Namespace: p.Namespace,
		Platform:  p.Name,
	}
	pf, err := c.PortForward(ctx, pfReq)
	if errors.Is(err, kubernetes.ErrComponentNotReady) && cfg.Flags.WaitTime > 0 {
		log.Warn("No httpsrv pod is available.")
		log.Info("Waiting for httpsrv pod to become available...")

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Flags.WaitTime)
		defer cancel()

		err = c.WaitPodReady(ctx, p, "httpsrv", "")
		if err == nil {
			pf, err = c.PortForward(ctx, pfReq)
		}
	}
	if err != nil {
		log.Fatal("Error starting proxy: %v", err)
	}

	return pf
}
