package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/api/admin"
	"github.com/xigxog/kubefox/libs/core/api/admin/v1alpha1"
	"github.com/xigxog/kubefox/libs/core/api/common"
	"github.com/xigxog/kubefox/libs/core/api/maker"
	"github.com/xigxog/kubefox/libs/core/api/uri"
	"sigs.k8s.io/yaml"
)

type resHeader struct {
	Kind       string      `json:"kind"`
	APIVersion string      `json:"apiVersion"`
	Metadata   resMetadata `json:"metadata"`
}

type resMetadata struct {
	Name string `json:"name"`
}

func pwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting working dir: %v", err)
	}

	return filepath.Clean(wd)
}

func getResURI(args []string) uri.URI {
	if len(args) == 0 {
		log.Fatal("No resource provided")
	}

	str := fmt.Sprintf("%s://%s/%s", uri.KubeFoxScheme, cfg.GitHub.Org.Name, args[0])
	u, err := uri.Parse(str)
	if err != nil {
		log.Fatal("Error generating URI for provided resource '%s': %v", args[0], err)
	}

	return u
}

func getOutFormat() string {
	if flags.OutFormat != "" {
		if strings.EqualFold(flags.OutFormat, "yaml") || strings.EqualFold(flags.OutFormat, "yml") {
			return "yaml"
		} else if strings.EqualFold(flags.OutFormat, "json") {
			return "json"
		} else {
			log.Fatal("Invalid output format '%s', provide one of: 'json', 'yaml'", flags.OutFormat)
		}
	}

	// default
	return "json"
}

func registerSystem() {
	platURI, err := uri.New(cfg.GitHub.Org.Name, uri.Platform, cfg.KubeFox.Platform)
	if err != nil {
		log.Fatal("Platform name is invalid: %v", err)
	}
	sysURI, err := uri.New(cfg.GitHub.Org.Name, uri.System, flags.System)
	if err != nil {
		log.Fatal("System name is invalid: %v", err)
	}

	ips := base64.StdEncoding.EncodeToString([]byte("kubefox:" + cfg.GitHub.Token))
	platform := maker.Empty[v1alpha1.Platform]()
	platform.SetName(cfg.KubeFox.Platform)
	// platform.SetOrganization(cfg.GitHub.Org.Name)
	platform.Systems = map[uri.Key]*common.PlatformSystem{
		uri.Key(sysURI.Name()): {
			ImagePullSecret: ips,
		},
	}

	log.Verbose("Registering system with the KubeFox platform")
	log.VerboseResp(admCli.Patch(platURI, platform))
}

func getObjFromFile(file string) (uri.URI, any) {
	var err error

	path, err := filepath.Abs(file)
	if err != nil {
		log.Fatal("Error resolving path to file '%s': %v", file, err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		log.Fatal("Error reading file '%s': %v", path, err)
	}

	// unmarshal base to get resource kind
	header := &resHeader{}
	unmarshal(path, contents, header)

	var u uri.URI
	kind := uri.KindFromString(header.Kind)
	subKind := uri.SubKindFromString(header.Kind)

	var res any
	if subKind == uri.Deployment || subKind == uri.Release {
		u, err = uri.New(cfg.GitHub.Org.Name, uri.Platform, cfg.KubeFox.Platform, subKind)
		res = newSubResource(u)
	} else {
		u, err = uri.New(cfg.GitHub.Org.Name, kind, header.Metadata.Name)
		res = newResource(u)
	}
	if err != nil {
		log.Fatal("Error generating resource URI: %v", err)
	}

	// unmarshal full to get all fields
	unmarshal(path, contents, res)
	if r, ok := res.(admin.SubObject); ok {
		if u, err = r.GetURI(cfg.GitHub.Org.Name, cfg.KubeFox.Platform); err != nil {
			log.Fatal("Error generating resource URI: %v", err)
		}
	}

	return u, res
}

func newResource(u uri.URI) any {
	r := maker.ObjectFromURI(u)
	if r == nil {
		log.Fatal("Unknown kind '%s' provided", u.Kind)
	}
	return r
}

func newSubResource(u uri.URI) any {
	r := maker.SubObjFromURI(u)
	if r == nil {
		log.Fatal("Unknown subresource kind '%s' provided", u.SubKind())
	}
	return r
}

func unmarshal(path string, contents []byte, res any) {
	var err error
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yml":
		fallthrough
	case ".yaml":
		err = yaml.Unmarshal(contents, res)
	case ".json":
		fallthrough
	default:
		err = json.Unmarshal(contents, res)
	}
	if err != nil {
		log.Fatal("Error unmarshaling file '%s': %v", path, err)
	}
}
