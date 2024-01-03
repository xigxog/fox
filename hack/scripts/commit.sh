#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

${SCRIPTS}/clean.sh

mkdir -p ${TOOLS_DIR}

# Ensure all source files have copyright header.
go install github.com/hashicorp/copywrite@v0.18.0
${TOOLS_DIR}/copywrite license

go mod tidy
go fmt ./...
go vet ./...

${SCRIPTS}/hello-world.sh
${SCRIPTS}/docs.sh

git add .
git commit
