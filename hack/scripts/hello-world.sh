#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

HELLO_WORLD_SRC="efs/hello-world"

rm -rf "${HELLO_WORLD_SRC}"
cp -r "${KUBEFOX_SRC}/examples/hello-world/kubefox" "${HELLO_WORLD_SRC}"
mv "${HELLO_WORLD_SRC}/go.mod" "${HELLO_WORLD_SRC}/go.mod.trim"
