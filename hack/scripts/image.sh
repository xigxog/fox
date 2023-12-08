#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

${SCRIPTS}/build.sh

if ${DEBUG}; then
	DOCKERFILE="--file Dockerfile.debug ."
fi

buildah bud --build-arg BIN="${BUILD_OUT}/${BIN}" --build-arg COMPRESS="${COMPRESS}" --tag "${IMAGE}" ${DOCKERFILE}

if ${PUSH}; then
	buildah push "${IMAGE}"
fi
