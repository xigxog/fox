#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

${SCRIPTS}/clean.sh

go fmt ./...
go vet ./...

${SCRIPTS}/hello-world.sh
${SCRIPTS}/docs.sh

git add .
git commit
