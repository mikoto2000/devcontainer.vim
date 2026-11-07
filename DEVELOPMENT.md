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

