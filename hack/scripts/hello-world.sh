#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

HELLO_WORLD_SRC="efs/hello-world"

rm -rf "${HELLO_WORLD_SRC}"
cp -r "${KUBEFOX_SRC}/examples/go/hello-world/kubefox" "${HELLO_WORLD_SRC}"
(
    cd "${HELLO_WORLD_SRC}"
    go mod init github.com/xigxog/kubefox/quickstart
    go mod tidy
    # Remove patch version from Go version.
    sed -i '/go 1.21/c\go 1.21' go.mod
)

# Go will not embed directories containing a go.mod file. To resolve this the
# extension .trim is added. This will be removed when Fox writes the template
# files to disk.
mv "${HELLO_WORLD_SRC}/go.mod" "${HELLO_WORLD_SRC}/go.mod.trim"
