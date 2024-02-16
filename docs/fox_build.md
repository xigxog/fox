## fox build

Build and optionally push an OCI image of component

### Synopsis

The build command will use Docker to build the specified component. By default
components are built using a KubeFox defined Dockerfile. A custom Dockerfile can
be provided my placing it in the root directory of the component. Please note
that the build working directory is the root of the repository, not the
component directory.

```
fox build <NAME> [flags]
```

### Examples

```
# Build and push OCI image for my-component.
fox build my-component --publish
```

### Options

```
      --force         force build even if component image exists
  -h, --help          help for build
  -k, --kind string   if provided the built image will be loaded into the kind cluster
      --no-cache      do not use cache when building image
      --push          publish image to OCI image registry
```

### Options inherited from parent commands

```
  -a, --app string                 path to directory containing KubeFox App
  -i, --info                       enable info output
  -o, --output string              output format, one of ["json", "yaml"] (default "yaml")
      --registry-address string    address of your container registry
      --registry-token string      access token for your container registry
      --registry-username string   username for your container registry
  -m, --timeout duration           timeout for command (default 5m0s)
  -v, --verbose                    enable verbose output
```

### SEE ALSO

* [fox](fox.md)	 - CLI for interacting with KubeFox

