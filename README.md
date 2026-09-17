# Pwndrop

Pwndrop is a self-hosted file-sharing service with a small web admin panel and
HTTP/WebDAV delivery. This fork provides a single-port Docker Compose deployment.

> Use it only for files and infrastructure you are authorized to host and test.

## Quick start

Prerequisites: Docker Engine with Docker Compose v2, `make`, and `openssl`.

```sh
git clone https://github.com/umbra2728/pwndrop.git
cd pwndrop
make init
make up
```

`make init` creates **one file only**: `.env`. It contains every runtime
setting plus a random admin password and secret admin URL, and is created with
mode `0600`. Save the generated credentials in a password manager.

Open the secret URL printed by `make init`:

```text
http://SERVER_HOST:PORT/<secret-admin-path>
```

Then sign in with the generated administrator credentials.

Useful commands:

```sh
make logs          # follow service logs
make down          # stop the service
make config-check  # validate Compose configuration
make test          # run Go tests
```

## Configuration: `.env` only

There is no TOML, INI, or mounted configuration file. Copy `.env.example` or
run `make init`, then edit `.env` before starting the service.

```dotenv
# The host and container port are both controlled by this one value.
PWN_DROP_HTTP_PORT=8080

# The Compose deployment is HTTP-only. TLS belongs to your reverse proxy.
PWN_DROP_HTTPS_PORT=0

# Persistent data volume and bundled UI paths.
PWN_DROP_DATA_DIR=/data
PWN_DROP_ADMIN_DIR=/app/admin
```

`compose.yaml` publishes exactly this mapping:

```yaml
ports:
  - "${PWN_DROP_HTTP_PORT}:${PWN_DROP_HTTP_PORT}"
```

For example, set `PWN_DROP_HTTP_PORT=8095`, run `make up`, and the app listens
and is published at `http://HOST:8095`.

The `PWN_DROP_SETUP_*` values initialize the first administrator account and
secret path only when the data volume has no users. They cannot reset a running
instance just because it restarts.

## Networking and security

The Compose stack deliberately owns only the configured HTTP port. TLS, public
domains, DNS, and certificate management are outside Pwndrop's responsibility.
Place Traefik, Caddy, Nginx, or another reverse proxy in front when HTTPS is
needed. Do not expose plain HTTP directly to the Internet unless that is an
intentional choice.

The container uses a named `pwndrop-data` volume, a read-only root filesystem,
a health endpoint at `/healthz`, a non-root application user, and dropped Linux
capabilities after startup.

## Data and backups

Uploads and the BoltDB database are in the `pwndrop-data` volume. Back it up:

```sh
docker run --rm \
  -v pwndrop-data:/data:ro \
  -v "$PWD":/backup \
  alpine tar czf /backup/pwndrop-data-backup.tgz -C /data .
```

To reset Pwndrop completely, run `make down`, remove `pwndrop-data`, and remove
`.env` before executing `make init` again. This permanently deletes all files
and accounts.

## Development

```sh
make build
make test
```

The backend is Go. The browser UI uses vendored Vue/Bootstrap assets under
`www/`, so no Node.js build step is required.

## License

This is a derivative of [kgretzky/pwndrop](https://github.com/kgretzky/pwndrop),
which is licensed under GPL-3.0. See [LICENSE](LICENSE).
