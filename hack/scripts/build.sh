#!/bin/bash
# Copyright 2023 XigXog
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.
#
# SPDX-License-Identifier: MPL-2.0


source "$(dirname "${BASH_SOURCE[0]}")/setup.sh"

mkdir -p "${BUILD_OUT}"

go build \
	-o "${BUILD_OUT}/${BIN}" \
	-ldflags " \
		-w -s
		-X github.com/xigxog/kubefox/build.date=${BUILD_DATE}\
		-X github.com/xigxog/kubefox/build.component=${COMPONENT} \
		-X github.com/xigxog/kubefox/build.hash=${COMPONENT_COMMIT} \
		-X github.com/xigxog/kubefox/build.rootCommit=${ROOT_COMMIT} \
		-X github.com/xigxog/kubefox/build.headRef=${HEAD_REF} \
		-X github.com/xigxog/kubefox/build.tagRef=${TAG_REF}" \
	main.go
