## Build time
FROM golang:1.21 as builder

ARG COMPONENT_DIR
ARG COMPONENT
ARG GIT_COMMIT
ARG GIT_REF

WORKDIR /app

# Cache dependencies.
COPY $COMPONENT_DIR/go.mod $COMPONENT_DIR/go.sum $COMPONENT_DIR/
RUN cd "$COMPONENT_DIR/" && go mod download

COPY ./ ./
RUN CGO_ENABLED=0 go build \
    -C "$COMPONENT_DIR" \
    -o /component \
    -ldflags " \
    -X github.com/xigxog/kubefox/libs/core/kubefox.ComponentName=$COMPONENT \
    -X github.com/xigxog/kubefox/libs/core/kubefox.GitCommit=$GIT_COMMIT \
    -X github.com/xigxog/kubefox/libs/core/kubefox.GitRef=$GIT_REF"

## Runtime
FROM ghcr.io/xigxog/base
COPY --from=builder /component /component
ENTRYPOINT [ "/component" ]
