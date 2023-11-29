# Compress binary
FROM ghcr.io/xigxog/upx:4.2.0 AS upx

ARG COMPRESS=false

COPY ./bin/fox /fox
RUN if ${COMPRESS}; then upx /fox; fi

# Runtime
FROM ghcr.io/xigxog/base
COPY --from=upx /fox /fox
ENTRYPOINT [ "/fox" ]
