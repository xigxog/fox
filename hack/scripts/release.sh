#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

OSES=(darwin linux windows)

for os in "${OSES[@]}"; do
    export GOOS=$os
    ${SCRIPTS}/package.sh &
done

wait
