#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

mkdir -p "${BUILD_OUT}"

go build \
	-o "${BUILD_OUT}/${BIN}" \
	-ldflags " \
		-w -s
		-X github.com/xigxog/kubefox/build.date=${BUILD_DATE}\
		-X github.com/xigxog/kubefox/build.component=${COMPONENT} \
		-X github.com/xigxog/kubefox/build.commit=${COMPONENT_COMMIT} \
		-X github.com/xigxog/kubefox/build.rootCommit=${ROOT_COMMIT} \
		-X github.com/xigxog/kubefox/build.headRef=${HEAD_REF} \
		-X github.com/xigxog/kubefox/build.tagRef=${TAG_REF}" \
	main.go
