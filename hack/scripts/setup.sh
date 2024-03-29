#!/bin/bash
# Copyright 2023 XigXog
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.
#
# SPDX-License-Identifier: MPL-2.0


set -o errexit

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." &>/dev/null && pwd -P)"
cd "${REPO_ROOT}"

SCRIPTS="hack/scripts"

KUBEFOX_SRC=${KUBEFOX_SRC:-"../kubefox"}
TOOLS_DIR="tools"

COMPONENT="fox"
COMPONENT_COMMIT=$(git rev-parse HEAD)
ROOT_COMMIT=${COMPONENT_COMMIT}

HEAD_REF=$(git symbolic-ref -q HEAD || true)
TAG_REF=$(git describe --tags --exact-match 2>/dev/null | xargs -I % echo "refs/tags/%")

CONTAINER_REGISTRY=${CONTAINER_REGISTRY:-"ghcr.io/xigxog"}
IMAGE_TAG=${IMAGE_TAG:-$(git symbolic-ref -q --short HEAD || git describe --tags --exact-match)}
IMAGE="${CONTAINER_REGISTRY}/fox:${IMAGE_TAG}"

BUILD_DATE=$(TZ=UTC date --iso-8601=seconds)

export GO111MODULE=on
export CGO_ENABLED=0
export GOARCH=amd64
export GOOS=${GOOS:-"linux"}
export GOBIN="${REPO_ROOT}/${TOOLS_DIR}"
export PATH="${PATH}:${GOBIN}"

BIN="fox"
TAR="${BIN}-${GOOS}-${GOARCH}.tar.gz"
BUILD_OUT_ROOT="bin"
BUILD_OUT="${BUILD_OUT_ROOT}/${GOOS}"
RELEASE_OUT="release"

COMPRESS=${COMPRESS:-false}
DEBUG=${DEBUG:-false}
DOCKERFILE=""
PUSH=${PUSH:-false}

if [ "$GOOS" == "windows" ]; then
    BIN="fox.exe"
fi

set -o pipefail -o xtrace -o nounset
