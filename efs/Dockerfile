# Copyright 2023 XigXog
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.
#
# SPDX-License-Identifier: MPL-2.0

## Build time
FROM golang:1.21 as builder

ARG APP_YAML
ARG BUILD_DATE
ARG COMPONENT_DIR
ARG COMPONENT
ARG COMPONENT_COMMIT
ARG ROOT_COMMIT
ARG HEAD_REF
ARG TAG_REF

WORKDIR /app

# Cache dependencies.
COPY ${APP_YAML} go.mo[d] go.su[m] ./
RUN go mod download || true

COPY ./ ./
RUN CGO_ENABLED=0 go build \
    -C "$COMPONENT_DIR" \
    -o /component \
    -ldflags " \
    -X github.com/xigxog/kubefox/build.date=${BUILD_DATE}\
    -X github.com/xigxog/kubefox/build.component=${COMPONENT} \
    -X github.com/xigxog/kubefox/build.commit=${COMPONENT_COMMIT} \
    -X github.com/xigxog/kubefox/build.rootCommit=${ROOT_COMMIT} \
    -X github.com/xigxog/kubefox/build.headRef=${HEAD_REF} \
    -X github.com/xigxog/kubefox/build.tagRef=${TAG_REF}"

## Runtime
FROM ghcr.io/xigxog/base
COPY --from=builder /component /component
ENTRYPOINT [ "/component" ]
