#!/bin/sh
# Bind-mounted host directories (e.g. ./pb_data in docker-compose.yml) are
# typically created root-owned, which the image's nonroot (65532) user can't
# write to. When started as root, fix ownership once and drop to nonroot
# before exec'ing the app; when already started as nonroot (e.g. a Kubernetes
# securityContext), just run directly.
set -e

if [ "$(id -u)" = "0" ]; then
    mkdir -p /app/pb_data
    chown -R 65532:65532 /app/pb_data
    exec su -s /bin/sh nonroot -c 'exec /app/subflow "$@"' -- subflow "$@"
fi

exec /app/subflow "$@"
