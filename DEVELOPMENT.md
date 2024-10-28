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

## Start dev container without devcontainer.vim

```sh
docker run -it --rm -v "$(pwd):/work" --workdir /work --name devcontainer.vim golang:1.22.1-bookworm
```

## Create project

```sh
go mod init github.com/mikoto2000/devcontainer.vim
```

## Generate and Use Completion Scripts

### Bash

```sh
source <(devcontainer.vim completion bash)
```

### Zsh

```sh
source <(devcontainer.vim completion zsh)
```

### Fish

```sh
devcontainer.vim completion fish | source
```
