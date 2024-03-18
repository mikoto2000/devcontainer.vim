# Development

## Start container

```sh
docker run -it --rm -v "$(pwd):/work" --workdir /work --name devcontainer.vim golang:1.22.1-bookworm
```


## Create project

```sh
go mod init github.com/mikoto2000/devcontainer.vim
```


## Format source

```sh
go fmt ./...
```

## Build

```sh
make build-all
```

