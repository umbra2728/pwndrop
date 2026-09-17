# Pwndrop

Pwndrop is a self-hosted file-sharing service with a small web admin panel and
HTTP/WebDAV delivery. This fork packages the application for a predictable,
single-port Docker Compose deployment.

> Use only for files and infrastructure you are authorized to host and test.

## What this fork changes

- Docker multi-stage build with a minimal runtime image.
- Docker Compose deployment exposing **only HTTP port 8080**.
- Persistent application data in the named `pwndrop-data` Docker volume.
- A readable `config.toml` runtime configuration instead of `pwndrop.ini`.
- `make init` creates a private `.env` with a random administrator password and
  secret admin URL. Secrets are not committed.
- Container healthcheck at `/healthz`, read-only root filesystem, dropped Linux
  capabilities, `no-new-privileges`, and a non-root application user.
- TLS and DNS are deliberately disabled in the supplied Compose deployment.
  Put a reverse proxy in front when HTTPS or a public domain is required.

## Quick start

Prerequisites: Docker Engine with Docker Compose v2, `make`, and `openssl`.

```sh
git clone https://github.com/umbra2728/pwndrop.git
cd pwndrop
make init
make up
```

`make init` prints the generated administrator password and secret admin path
once, and stores them in `.env` with mode `0600`. Save those values in a secure
password manager. Then open:

```text
http://SERVER_HOST:8080/<secret-admin-path>
```

The first visit to that secret path grants access to the login screen. Sign in
with the generated credentials.

Useful commands:

```sh
make logs          # follow service logs
make down          # stop the service
make config-check  # validate Compose interpolation and structure
make test          # run Go tests
```

## Configuration

`make init` creates two local files:

| File | Purpose | Commit it? |
| --- | --- | --- |
| `config.toml` | Network, paths, and listener configuration | No; use `config.example.toml` as the template |
| `.env` | Initial administrator credentials and secret path | **Never** |

The supplied `config.toml` uses:

```toml
[pwndrop]
listen_ip = "0.0.0.0"
http_port = 8080
https_port = 0
data_dir = "/data"
admin_dir = "/app/admin"
```

The environment values `PWN_DROP_SETUP_*` are bootstrap-only: they are applied
only when the data volume has no administrator account. This prevents a
container restart from rotating the secret path or resetting a password.

### HTTPS and public access

This Compose stack owns only port `8080` and serves plain HTTP. Terminate TLS
in an external reverse proxy (Traefik, Caddy, Nginx, etc.) and proxy requests to
`http://HOST:8080`. Do not expose the Pwndrop port directly to the Internet
unless you explicitly accept plaintext HTTP.

The old built-in DNS server and automatic ACME/Let's Encrypt mode remain in the
application for manual deployments, but are off in this Compose setup.

## Data and backups

All uploads, the BoltDB database, and any certificates used by a manual setup
are stored in the `pwndrop-data` Docker volume. Back it up before upgrades:

```sh
docker run --rm \
  -v pwndrop-data:/data:ro \
  -v "$PWD":/backup \
  alpine tar czf /backup/pwndrop-data-backup.tgz -C /data .
```

To reset the instance completely, stop the stack, remove `pwndrop-data`, and
run `make init` again after deliberately removing the local `.env` and
`config.toml` files. This permanently deletes all uploaded files and accounts.

## Development

```sh
make build
make test
```

The backend is Go and the browser UI is vendored Vue/Bootstrap assets under
`www/`; no Node.js build step is required. Go dependencies are vendored, so the
normal build uses `-mod=vendor`.

## Original project and license

This is a private derivative of [kgretzky/pwndrop](https://github.com/kgretzky/pwndrop).
The original project is licensed under GPL-3.0; see [LICENSE](LICENSE).
