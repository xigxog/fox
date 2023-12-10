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

# TODO deal nested app repos

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
