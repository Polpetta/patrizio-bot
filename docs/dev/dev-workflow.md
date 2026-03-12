# Development Workflow

The Makefile contains a few handy targets that cover everything you need to compile, test and deploy Patrizio.

## Build

```bash
make build
```

This compiles the binary into `./bin/patrizio`.

## Run

```bash
make run
```

or

```bash
./bin/patrizio serve
```

The bot starts, pulls the Delta Chat RPC client, and begins listening for messages.

## Tests

```bash
make test
```

All Go tests run, covering the pure domain logic and the adapter integration.

## Docker

The Dockerfile is a multi‑stage build that ends up with a tiny, distroless image (`gcr.io/distroless/static‑debian12`).
The runtime image contains the compiled binary and the bundled `deltachat‑rpc‑server` from the upstream releases.

```bash
make docker-build
```

## Linting

Linting is enforced with `golangci‑lint` using the 120‑column limit defined in `.golangci.yml`.

```bash
make lint
```

## Pre‑commit Hooks

The project uses the `pre‑commit` framework. Hooks run linters,
formatters, tests, and even the Docker build automatically on each commit.

---

## File references

* `Makefile`
* `.golangci.yml`
* `.pre-commit-config.yaml`
