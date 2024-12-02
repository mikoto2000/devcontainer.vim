APP_NAME := devcontainer.vim
GOARCH_AMD64 := amd64
GOARCH_ARM64 := arm64
WINDOWS_BINARY_NAME := ${APP_NAME}-windows-${GOARCH_AMD64}.exe
LINUX_BINARY_NAME := ${APP_NAME}-linux-${GOARCH_AMD64}
DARWIN_BINARY_NAME := ${APP_NAME}-darwin-${GOARCH_ARM64}

GO_BIN := ${GOPATH}/bin
VERSION := 3.0.2
LD_FLAGS := "-s -w -X main.version=${VERSION}"

DEST := ./build

WATCH_SRC := ./main.go \
						 ./devcontainer/DevcontainerJson.go \
						 ./devcontainer/devcontainer.go \
						 ./devcontainer/run.go \
						 ./devcontainer/start.go \
						 ./devcontainer/dockerRunVimArgs_darwin_arm64.go \
						 ./devcontainer/dockerRunVimArgs_linux_amd64.go \
						 ./devcontainer/dockerRunVimArgs_windows_amd64.go \
						 ./devcontainer/devcontainerStartVimArgs_darwin_arm64.go \
						 ./devcontainer/devcontainerStartVimArgs_linux_amd64.go \
						 ./devcontainer/devcontainerStartVimArgs_windows_amd64.go \
						 ./devcontainer/readConfigurationResult.go \
						 ./devcontainer/upCommandResult.go \
						 ./docker/docker.go \
						 ./docker/dockerPsResult.go \
						 ./dockercompose/dockerCompose.go \
						 ./dockercompose/dockerComposePsResult.go \
						 ./tools/tools.go \
						 ./tools/vim_linux_amd64.go \
						 ./tools/vim_darwin_arm64.go \
						 ./tools/vim_windows_amd64.go \
						 ./tools/devcontainer.go \
						 ./tools/devcontainer_darwin.go \
						 ./tools/devcontainer_linux.go \
						 ./tools/devcontainer_windows.go \
						 ./tools/clipboard-data-receiver.go \
						 ./util/util.go

### 開発関連
# 開発環境の都合で、個別にビルドできるようにしている
# (Linux コンテナ上でコーディングを行い、 Windows 上で実行することがあるため)
all: build
build: build/devcontainer.vim
build/devcontainer.vim: ${WATCH_SRC}
	go build -ldflags=${LD_FLAGS} -trimpath -o ./build/${APP_NAME}

build-all: build-windows build-linux build-darwin

build-windows: build/${WINDOWS_BINARY_NAME}
build/${WINDOWS_BINARY_NAME}: ${WATCH_SRC}
	GOOS=windows GOARCH=${GOARCH_AMD64} go build -ldflags=${LD_FLAGS} -trimpath -o build/${WINDOWS_BINARY_NAME}

build-linux: build/${LINUX_BINARY_NAME}
build/${LINUX_BINARY_NAME}: ${WATCH_SRC}
	GOOS=linux GOARCH=${GOARCH_AMD64} go build -ldflags=${LD_FLAGS} -trimpath -o build/${LINUX_BINARY_NAME}

build-darwin: build/${DARWIN_BINARY_NAME}
build/${DARWIN_BINARY_NAME}: ${WATCH_SRC}
	GOOS=darwin GOARCH=${GOARCH_ARM64} go build -ldflags=${LD_FLAGS} -trimpath -o build/${DARWIN_BINARY_NAME}

.PHONY: lint
lint:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck -checks inherit,ST1003,ST1016 ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: clean
clean:
	rm -rf build

