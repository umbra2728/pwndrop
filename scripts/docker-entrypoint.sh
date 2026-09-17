#!/usr/bin/env sh
set -eu

# Docker creates named volumes as root. Prepare the persistent directory before
# dropping to the unprivileged application account.
mkdir -p /data
chown -R pwndrop:pwndrop /data

exec su-exec pwndrop:pwndrop /app/pwndrop "$@"
