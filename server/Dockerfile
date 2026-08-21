FROM alpine:3.22.1 AS pocketbase-download

ARG TARGETARCH
RUN apk add --no-cache ca-certificates unzip wget
WORKDIR /download
COPY pocketbase-version.env ./
RUN set -eu; \
    . ./pocketbase-version.env; \
    archive_arch="$TARGETARCH"; \
    if [ -z "$archive_arch" ]; then \
      case "$(uname -m)" in \
        x86_64) archive_arch=amd64 ;; \
        aarch64) archive_arch=arm64 ;; \
        *) printf 'Unsupported build architecture.\n' >&2; exit 64 ;; \
      esac; \
    fi; \
    case "$archive_arch" in \
      amd64) expected_sha256="$POCKETBASE_SHA256_LINUX_AMD64" ;; \
      arm64) expected_sha256="$POCKETBASE_SHA256_LINUX_ARM64" ;; \
      *) printf 'Unsupported build architecture: %s\n' "$archive_arch" >&2; exit 64 ;; \
    esac; \
    archive="pocketbase_${POCKETBASE_VERSION}_linux_${archive_arch}.zip"; \
    wget -q "https://github.com/pocketbase/pocketbase/releases/download/v${POCKETBASE_VERSION}/${archive}" -O pocketbase.zip; \
    printf '%s  %s\n' "$expected_sha256" pocketbase.zip | sha256sum -c -; \
    unzip -q pocketbase.zip pocketbase

FROM alpine:3.22.1

RUN apk add --no-cache ca-certificates su-exec \
    && addgroup -S -g 10001 pocketbase \
    && adduser -S -D -H -u 10001 -G pocketbase pocketbase \
    && mkdir -p /pb/pb_data \
    && chown -R pocketbase:pocketbase /pb

WORKDIR /pb
COPY --from=pocketbase-download --chown=pocketbase:pocketbase /download/pocketbase /pb/pocketbase
COPY --chown=pocketbase:pocketbase pb_hooks /pb/pb_hooks
COPY --chown=pocketbase:pocketbase pb_migrations /pb/pb_migrations
COPY --chown=pocketbase:pocketbase --chmod=0755 docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

EXPOSE 8090
VOLUME ["/pb/pb_data"]
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
  CMD wget -q -O /dev/null http://127.0.0.1:8090/api/health || exit 1
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["serve", "--http=0.0.0.0:8090", "--dir=/pb/pb_data", "--hooksDir=/pb/pb_hooks", "--migrationsDir=/pb/pb_migrations"]
