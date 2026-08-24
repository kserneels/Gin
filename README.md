# GinLog

A tiny, self-hosted app for keeping track of gin mixes you've enjoyed — the
gin, the tonic, the garnish, the ratio, where you had it, and how you'd rate
it.

Built to be as lightweight as possible to host: it's a single Go binary with
an embedded SQLite database and server-rendered HTML. No Node, no build
step, no external services — just one process and one file on disk.

## Features

- Add a mix with gin, tonic, garnish/fruit, ratio, glassware, where you had
  it, a 1-5 star rating, and free-form notes.
- Browse everything you've saved, newest first.
- Search across gin, tonic, garnish and location.
- Edit or delete any entry.
- Optional login (session cookie + bcrypt-hashed password) to protect the
  app when it's exposed to the internet.

## Running locally

Requires Go 1.24+.

```sh
go run .
```

The app listens on `:8080` by default and stores data in `./ginlog.db`
(created automatically on first run). Open http://localhost:8080.

Configuration is via environment variables:

| Variable             | Default     | Description                                                          |
|----------------------|-------------|------------------------------------------------------------------------|
| `ADDR`               | `:8080`     | Address/port to listen on                                             |
| `DB_PATH`            | `ginlog.db` | Path to the SQLite database file                                      |
| `AUTH_USERNAME`      | (unset)     | Login username. If unset, the app runs with no login page at all.     |
| `AUTH_PASSWORD`      | (unset)     | Login password, in plain text. Simplest option — see below.           |
| `AUTH_PASSWORD_HASH` | (unset)     | Bcrypt hash of the login password. Takes priority over `AUTH_PASSWORD` if both are set. |
| `INSECURE_COOKIE`    | (unset)     | Set to any value to allow the session cookie over plain HTTP (only needed for local testing without TLS). |

## Enabling login

Auth is off by default. To turn it on, set `AUTH_USERNAME` and one of the
two password options:

**Plain password (simplest):** set `AUTH_PASSWORD` to whatever you want.
In Portainer, add `AUTH_USERNAME` and `AUTH_PASSWORD` as stack environment
variables, same as you did for `HOST_PORT`.

**Bcrypt hash (avoids storing the password in plain text):** generate one
with the binary itself, no extra tools needed:

```sh
go run . hash 'your-password-here'
# or, with the Docker image:
docker run --rm ginlog hash 'your-password-here'
```

Set `AUTH_PASSWORD_HASH` to the printed value instead (quote it — it
contains `$` characters).

Either way, once `AUTH_USERNAME` and a password are set, every page
requires logging in, with a 30-day session cookie so you're not asked
repeatedly on your phone.

## Running with Docker

```sh
docker compose up -d --build
```

This builds a small Alpine-based image, runs it as a non-root user, and
persists the database in a Docker volume so it survives restarts. By default
it publishes on host port `8091` (the container itself always listens on
`8080` internally). To use a different host port, set `HOST_PORT`:

```sh
HOST_PORT=9091 docker compose up -d --build
```

In Portainer, set `HOST_PORT` as a stack environment variable instead of
editing the compose file.

Without compose:

```sh
docker build -t ginlog .
docker run -d -p 8091:8080 -v ginlog-data:/data ginlog
```

## Deploying elsewhere

Because it's a single statically-linkable Go binary plus one SQLite file,
GinLog runs anywhere you can run a small process: a cheap VPS behind a
reverse proxy (Caddy/nginx), a Raspberry Pi, Fly.io, Render, a systemd
service, etc. Build a binary for the target platform with:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ginlog .
```

then copy it and the app will create its SQLite database on first run. Back
up the whole app by copying the `ginlog.db` file.
