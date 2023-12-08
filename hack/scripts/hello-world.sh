#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

HELLO_WORLD_SRC="efs/hello-world"

rm -rf "${HELLO_WORLD_SRC}"
cp -r "${EXAMPLES_SRC}/go/hello-world/kubefox" "${HELLO_WORLD_SRC}"
mv "${HELLO_WORLD_SRC}/go.mod" "${HELLO_WORLD_SRC}/go.mod.trim"
