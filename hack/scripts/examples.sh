#!/bin/bash
# Copyright 2023 XigXog
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.
#
# SPDX-License-Identifier: MPL-2.0

# TODO clean this up

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

HELLO_WORLD_SRC="efs/hello-world"

rm -rf "${HELLO_WORLD_SRC}"
cp -r "${KUBEFOX_SRC}/examples/go/hello-world/kubefox" "${HELLO_WORLD_SRC}"
(
    cd "${HELLO_WORLD_SRC}"
    go mod init github.com/xigxog/kubefox/quickstart
    go mod tidy
    # Remove patch version from Go version.
    sed -i '/go 1.22/c\go 1.22' go.mod
)

GRAPHQL_SRC="efs/graphql"

rm -rf "${GRAPHQL_SRC}"
cp -r "${KUBEFOX_SRC}/examples/go/graphql" "${GRAPHQL_SRC}"
(
    cd "${GRAPHQL_SRC}"
    go mod init github.com/xigxog/kubefox/graphql
    go mod tidy
    # Remove patch version from Go version.
    sed -i '/go 1.22/c\go 1.22' go.mod
)

# Go will not embed directories containing a go.mod file. To resolve this the
# extension .trim is added. This will be removed when Fox writes the template
# files to disk.
mv "${HELLO_WORLD_SRC}/go.mod" "${HELLO_WORLD_SRC}/go.mod.trim"
mv "${GRAPHQL_SRC}/go.mod" "${GRAPHQL_SRC}/go.mod.trim"
