# Compress binary
FROM ghcr.io/xigxog/upx:4.2.1 AS upx

ARG BIN
ARG COMPRESS=false

COPY ${BIN} /fox
RUN if ${COMPRESS}; then upx /fox; fi

# Runtime
FROM ghcr.io/xigxog/base
COPY --from=upx /fox /fox
ENTRYPOINT [ "/fox" ]
