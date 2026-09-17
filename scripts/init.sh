#!/usr/bin/env sh
set -eu

ENV_FILE=${ENV_FILE:-.env}
CONFIG_FILE=${CONFIG_FILE:-config.toml}
ADMIN_USERNAME=${PWN_DROP_SETUP_USERNAME:-admin}

if [ -e "$ENV_FILE" ] || [ -e "$CONFIG_FILE" ]; then
    echo "Refusing to overwrite $ENV_FILE or $CONFIG_FILE." >&2
    echo "Remove them deliberately before initializing a new instance." >&2
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
cp config.example.toml "$CONFIG_FILE"
cat >"$ENV_FILE" <<EOF
PWN_DROP_SETUP_USERNAME=$ADMIN_USERNAME
PWN_DROP_SETUP_PASSWORD=$password
PWN_DROP_SETUP_SECRET_PATH=$secret_path
PWN_DROP_SETUP_REDIRECT_URL=
EOF
chmod 600 "$ENV_FILE" "$CONFIG_FILE"

cat <<EOF
Initialized Pwndrop configuration.

Administrator username: $ADMIN_USERNAME
Administrator password: $password
Secret admin URL path: $secret_path

These values are stored in $ENV_FILE (mode 600). Keep it private.
Start the service with: docker compose up -d --build
EOF
