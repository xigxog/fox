#!/bin/bash

set -e

git_commit=$(git log -n 1 --format="%h" -- ./)
git_ref=$(git symbolic-ref -q --short HEAD || git describe --tags --exact-match)

export GOOS=${1:-"linux"}
build_dir=${2:-"bin"}
rel_dir=${3:-"release"}

GIT_COMMIT=${GIT_COMMIT:-"$git_commit"}
GIT_REF=${GIT_REF:-"$git_ref"}

export CGO_ENABLED=0
export GOARCH=amd64

if [ -z "${GIT_COMMIT}" ]; then
    echo "Environment variable GIT_COMMIT must be set for release target"
    exit 1
fi
if [ -z "${GIT_REF}" ]; then
    echo "Environment variable GIT_REF must be set for release target"
    exit 1
fi

bin="fox"
tar="${bin}-$(basename ${GIT_REF})-${GOOS}-${GOARCH}.tar.gz"

if [ "$GOOS" == "windows" ]; then
    bin="fox.exe"
fi

echo "Creating 🦊 Fox release package..."
echo "Git Commit: $GIT_COMMIT, Git Ref: $GIT_REF, Package: ${rel_dir}/${tar}"

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

echo "Fin."