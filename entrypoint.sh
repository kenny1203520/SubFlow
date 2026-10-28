#!/bin/sh
# Persistent PocketBase data must be writable by the image's default UID.
# Ownership changes are an explicit one-time operator action, never part of
# normal container startup.
set -e

if [ "${1:-}" = "init-permissions" ]; then
    if [ "$(id -u)" != "0" ]; then
        echo "init-permissions must be run explicitly as root (docker run --user 0:0 ... init-permissions)" >&2
        exit 1
    fi
    mkdir -p /app/pb_data
    chown -R 65532:65532 /app/pb_data
    exit 0
fi

if [ "$(id -u)" = "0" ]; then
    echo "Refusing to run SubFlow as root; use the default image user 65532:65532" >&2
    exit 1
fi

if [ ! -w /app/pb_data ]; then
    echo "PocketBase data directory is not writable by UID $(id -u); stop the container and run init-permissions once as root" >&2
    exit 1
fi

exec /app/subflow "$@"
