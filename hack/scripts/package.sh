#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

${SCRIPTS}/clean.sh
mkdir -p ${RELEASE_OUT}

${SCRIPTS}/build.sh

tar -czvf "${RELEASE_OUT}/${TAR}" --transform='s,.*/,,' "${BUILD_OUT}/${BIN}" LICENSE README.md 1>/dev/null
(
    cd "${RELEASE_OUT}"
    sha256sum "${TAR}" >"${TAR}.sha256sum"
)
