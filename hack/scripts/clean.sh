#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

rm -rf ${BUILD_OUT_ROOT} ${RELEASE_OUT} ${TOOLS_DIR}
