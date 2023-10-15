#!/bin/bash

set -e

export CGO_ENABLED=0
export GOARCH=amd64
export GOOS=${1:-"linux"}
build_dir=${2:-"bin"}
rel_dir=${3:-"release"}
bin="fox"
tar="${bin}-${GIT_REF}-${GOOS}-${GOARCH}.tar.gz"

if [ "$GOOS" == "windows" ]; then
    bin="fox.exe"
fi

if [ -z "${GIT_REF}" ]; then
    echo "Environment variable GIT_REF must be set for release target"
    exit 1
fi

go build \
    -o "${build_dir}/${bin}" \
    -ldflags " \
    -X github.com/xigxog/kubefox/libs/core/kubefox.GitCommit=$GIT_COMMIT \
    -X github.com/xigxog/kubefox/libs/core/kubefox.GitRef=$GIT_REF" \
    main.go

mkdir -p "${rel_dir}"
tar -czvf "${rel_dir}/${tar}" --transform='s,.*/,,' "${build_dir}/${bin}" LICENSE README.md 1>/dev/null
(
    cd "${rel_dir}"
    sha256sum "${tar}" >"${tar}.sha256sum"
)
