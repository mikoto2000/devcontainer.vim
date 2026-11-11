# Development

## Start dev container

```sh
devcontainer.vim start .
```

## Format source

```sh
go fmt ./...
```

## Build

```sh
make build-all
```

## Lint

Install the pinned Staticcheck release with `make install-lint-tools` and ensure your Go binary directory is on `PATH`. Run `make lint` to use the installed tool.

## Package responsibilities

The repository builds one application and uses one Go module. The CLI selects the requested features and delegates container lifecycle operations to `devcontainer`. That package coordinates the `docker`, `dockercompose`, and `tools` adapters. Workspace configuration and cleanup belong to `devcontainer`; CLI handlers use its result instead of deleting workspace caches themselves.

The internal packages keep independent operations separate.

- `internal/host` handles local executable lookup, file permissions, and host platform detection.
- `internal/configstore` owns configuration directories, workspace cache keys, and JSONC merging.
- `internal/githubrelease` handles GitHub release metadata and authentication.
- `internal/portforward` relays TCP traffic with cancellation, independently of Docker.

These packages do not depend on the CLI, container orchestration, or each other. Production callers import the package that owns the operation. The public `util` package delegates to these implementations to preserve compatibility for existing v3 callers.

## Start dev container without devcontainer.vim

```sh
docker run -it --rm -v "$(pwd):/work" --workdir /work --name devcontainer.vim golang:1.26.0-bookworm
```

## Create project

```sh
go mod init github.com/mikoto2000/devcontainer.vim
```

## Update major version

```sh
go mod edit -module github.com/mikoto2000/devcontainer.vim/v3
```

