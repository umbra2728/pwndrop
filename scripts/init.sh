#!/usr/bin/env sh
set -eu

ENV_FILE=${ENV_FILE:-.env}
ADMIN_USERNAME=${PWN_DROP_SETUP_USERNAME:-admin}
HTTP_PORT=${PWN_DROP_HTTP_PORT:-8080}

if [ -e "$ENV_FILE" ]; then
    echo "Refusing to overwrite $ENV_FILE." >&2
    echo "Remove it deliberately before initializing a new instance." >&2
    exit 1
fi

if ! command -v openssl >/dev/null 2>&1; then
    echo "openssl is required to generate initial credentials." >&2
    exit 1
fi

random_value() {
    openssl rand -base64 36 | tr -dc 'A-Za-z0-9' | cut -c1-32
}

password=$(random_value)
secret_path="/$(random_value)"

umask 077
cat >"$ENV_FILE" <<EOF
# Pwndrop Compose configuration. This is the only configuration file.
PWN_DROP_LISTEN_IP=0.0.0.0
PWN_DROP_HTTP_PORT=$HTTP_PORT
PWN_DROP_HTTPS_PORT=0
PWN_DROP_DATA_DIR=/data
PWN_DROP_ADMIN_DIR=/app/admin

PWN_DROP_SETUP_USERNAME=$ADMIN_USERNAME
PWN_DROP_SETUP_PASSWORD=$password
PWN_DROP_SETUP_SECRET_PATH=$secret_path
PWN_DROP_SETUP_REDIRECT_URL=
EOF
chmod 600 "$ENV_FILE"

cat <<EOF
Initialized Pwndrop.

Published HTTP port: $HTTP_PORT
Administrator username: $ADMIN_USERNAME
Administrator password: $password
Secret admin URL path: $secret_path

All configuration is in $ENV_FILE (mode 600). Keep it private.
Start the service with: docker compose up -d --build
EOF
