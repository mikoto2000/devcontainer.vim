APP_NAME := devcontainer.vim
GOARCH := amd64
WINDOWS_BINARY_NAME := ${APP_NAME}-windows-${GOARCH}.exe
LINUX_BINARY_NAME := ${APP_NAME}-linux-${GOARCH}
DARWIN_BINARY_NAME := ${APP_NAME}-darwin-${GOARCH}

GO_BIN := ${GOPATH}/bin
LD_FLAGS := "-s -w"
VERSION := 0.0.1

DEST := ./build

SRC := ./main.go

### 開発関連
# 開発環境の都合で、個別にビルドできるようにしている
# (Linux コンテナ上でコーディングを行い、 Windows 上で実行することがあるため)
all: build
build: build/devcontainer.vim
build/devcontainer.vim: ${SRC}
	go build -ldflags=${LD_FLAGS} -trimpath -o ./build/${APP_NAME} ${SRC}

build-all: build-windows build-linux build-darwin

build-windows: build/${WINDOWS_BINARY_NAME}
build/${WINDOWS_BINARY_NAME}: ./${SRC}
	GOOS=windows GOARCH=${GOARCH} go build -ldflags=${LD_FLAGS} -trimpath -o build/${WINDOWS_BINARY_NAME} ${SRC}

build-linux: build/${LINUX_BINARY_NAME}
build/${LINUX_BINARY_NAME}: main.go
	GOOS=linux GOARCH=${GOARCH} go build -ldflags=${LD_FLAGS} -trimpath -o build/${LINUX_BINARY_NAME} ${SRC}

build-darwin: build/${DARWIN_BINARY_NAME}
build/${DARWIN_BINARY_NAME}: main.go
	GOOS=darwin GOARCH=${GOARCH} go build -ldflags=${LD_FLAGS} -trimpath -o build/${DARWIN_BINARY_NAME} ${SRC}

.PHONY: clean
clean:
	rm -rf build

