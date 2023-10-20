package repo

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/archive"
	"github.com/moby/patternmatcher/ignorefile"
	"github.com/xigxog/fox/efs"
	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/utils"
	"github.com/xigxog/kubefox/libs/core/kubefox"
)

const (
	injectedDockerfile = "__Dockerfile"
)

type DockerfileTar struct {
	dockerfile []byte
	wrapped    io.ReadCloser
	read       int
}

func (r *repo) Build(compDirName string) string {
	img := r.GetCompImageFromDir(compDirName)
	compDir := filepath.Join(ComponentsDirName, compDirName)
	compName := utils.Clean(compDirName)
	gitCommit := r.GetCompCommit(compDirName)
	gitRef := r.GetRefName()
	regAuth := r.GetRegAuth()

	if !(r.cfg.Flags.ForceBuild || r.cfg.Flags.NoCache) {
		if found, _ := r.ensureImageExists(img, false); found {
			log.Info("Component image '%s' exists, skipping build.", img)
			r.KindLoad(img)
			log.InfoNewline()
			return img
		}
	}

	log.Info("Building component image '%s'.", img)
	dfPath := filepath.Join(r.cfg.Flags.RepoPath, ComponentsDirName, compDirName, "Dockerfile")
	df, err := os.ReadFile(dfPath)
	if err != nil {
		log.Verbose("Using default Dockerfile for build")
		df, _ = efs.EFS.ReadFile("Dockerfile")
	} else {
		log.Verbose("Using custom Dockerfile '%s' for build", dfPath)
	}

	dfi, err := NewDFI(r.path, df)
	if err != nil {
		log.Fatal("Error creating container tar: %v", err)
	}
	labels := map[string]string{
		kubefox.LabelOCIComponent: compName,
		kubefox.LabelOCICreated:   time.Now().Format(time.RFC3339),
		kubefox.LabelOCIRevision:  gitCommit,
		kubefox.LabelOCISource:    r.GetRepoURL(),
	}

	buildResp, err := r.docker.ImageBuild(context.Background(), dfi, types.ImageBuildOptions{
		Dockerfile: injectedDockerfile,
		NoCache:    r.cfg.Flags.NoCache,
		Remove:     true,
		Tags:       []string{img},
		Labels:     labels,
		BuildArgs: map[string]*string{
			"COMPONENT":     &compName,
			"COMPONENT_DIR": &compDir,
			"GIT_COMMIT":    &gitCommit,
			"GIT_REF":       &gitRef,
		},
	})
	if err != nil {
		log.Fatal("Error building container image: %v", err)
	}
	logResp(buildResp.Body, true)

	if r.cfg.IsLocalRegistry() {
		log.Verbose("Local registry is set, container image push will be skipped.")
	}
	if r.cfg.Flags.PushImage && !r.cfg.IsLocalRegistry() {
		log.Info("Pushing component image '%s'.", img)

		pushResp, err := r.docker.ImagePush(context.Background(), img, types.ImagePushOptions{
			RegistryAuth: regAuth,
		})
		if err != nil {
			log.Fatal("Error pushing container image: %v", err)
		}
		logResp(pushResp, true)
	}

	r.KindLoad(img)
	log.InfoNewline()
	return img
}

func (r *repo) ensureImageExists(img string, pull bool) (bool, error) {
	if r.cfg.IsLocalRegistry() {
		found := r.imageExistsLocal(img)
		if !found && pull {
			return false, fmt.Errorf("component image does not exist locally and no remote registry available")
		}

		return found, nil
	}

	if di, err := r.docker.DistributionInspect(context.Background(), img, r.GetRegAuth()); err != nil {
		log.Verbose("%s", err)
		return false, err

	} else {
		log.Verbose("Digest: %s", di.Descriptor.Digest)

		if pull && !r.imageExistsLocal(img) {
			pullResp, err := r.docker.ImagePull(context.Background(), img, types.ImagePullOptions{
				RegistryAuth: r.GetRegAuth(),
			})
			if err != nil {
				return false, fmt.Errorf("error pulling component image: %v", err)
			}
			if err := logResp(pullResp, false); err != nil {
				return false, fmt.Errorf("error pulling component image: %v", err)
			}
			return true, nil
		}
	}

	return true, nil
}

func (r *repo) imageExistsLocal(img string) bool {
	l, _ := r.docker.ImageList(context.Background(), types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", img)),
	})

	found := len(l) > 0
	if found {
		log.Verbose("Image '%s' found locally.", img)
	} else {
		log.Verbose("Image '%s' not found locally.", img)
	}

	return found
}

func (r *repo) KindLoad(img string) {
	kind := r.cfg.Flags.Kind
	if kind == "" && r.cfg.Kind.AlwaysLoad {
		kind = r.cfg.Kind.ClusterName
	}
	if kind == "" {
		return
	}

	log.Info("Loading component image '%s' into Kind cluster '%s'.", img, kind)
	if found, err := r.ensureImageExists(img, true); !found {
		if err != nil {
			log.Fatal("Error loading component image into Kind: %v", err)
		}
		log.Fatal("Component image does not exist, please build it first.")
	}

	cmd := exec.Command("kind", "load", "docker-image", "--name="+kind, img)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Error("%s", strings.TrimSpace(string(out)))
		log.Fatal("Error loading component image into Kind: %v", err)
	} else {
		log.Verbose("%s", strings.TrimSpace(string(out)))
	}
}

func (r *repo) GetRegAuth() string {
	token := r.cfg.ContainerRegistry.Token
	if r.cfg.GitHub.Token != "" {
		token = r.cfg.GitHub.Token
	}
	authCfg, _ := json.Marshal(registry.AuthConfig{
		Username: "kubefox",
		Password: token,
	})

	return base64.StdEncoding.EncodeToString(authCfg)
}

func logResp(resp io.ReadCloser, fatal bool) error {
	defer resp.Close()

	scanner := bufio.NewScanner(resp)
	for scanner.Scan() {
		l := make(map[string]any)
		json.Unmarshal(scanner.Bytes(), &l)
		logLine(l, "stream")
		logLine(l, "status", "id")
		if s, f := l["error"]; f {
			if fatal {
				log.Fatal("%s", s)
			} else {
				return fmt.Errorf("%s", s)
			}
		}
	}

	return nil
}

func logLine(l map[string]any, keys ...string) {
	var msg string
	for _, k := range keys {
		if s, f := l[k]; f {
			if msg == "" {
				msg = fmt.Sprintf("%s", s)
			} else {
				msg = fmt.Sprintf("%s %s", msg, s)
			}

		}
	}
	msg = strings.ReplaceAll(msg, "\n", "")
	if strings.TrimSpace(msg) != "" {
		log.Verbose("%s", msg)
	}
}

func NewDFI(path string, df []byte) (*DockerfileTar, error) {
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	w.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     injectedDockerfile,
		Size:     int64(len(df)),
		Mode:     644,
		ModTime:  time.Time{},
	})
	w.Write(df)
	w.Flush()

	dif, err := os.Open(filepath.Join(path, ".dockerignore"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	var ignore []string
	if dif != nil {
		ignore, err = ignorefile.ReadAll(dif)
		if err != nil {
			return nil, err
		}
	}
	tar, err := archive.TarWithOptions(path, &archive.TarOptions{
		ExcludePatterns: ignore,
	})
	if err != nil {
		return nil, err
	}

	return &DockerfileTar{
		wrapped:    tar,
		dockerfile: buf.Bytes(),
	}, nil
}

func (dfi *DockerfileTar) Read(p []byte) (n int, err error) {
	if dfi.read < len(dfi.dockerfile) {
		c := copy(p, dfi.dockerfile)
		dfi.read = dfi.read + c
		return c, nil
	}

	return dfi.wrapped.Read(p)
}

func (dfi *DockerfileTar) Close() error {
	return dfi.wrapped.Close()
}
