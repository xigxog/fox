#!/bin/bash
# Copyright 2023 XigXog
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.
#
# SPDX-License-Identifier: MPL-2.0


source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

${SCRIPTS}/clean.sh
mkdir -p ${RELEASE_OUT}

${SCRIPTS}/build.sh

tar -czvf "${RELEASE_OUT}/${TAR}" --transform='s,.*/,,' "${BUILD_OUT}/${BIN}" LICENSE README.md 1>/dev/null
(
    cd "${RELEASE_OUT}"
    sha256sum "${TAR}" >"${TAR}.sha256sum"
)
