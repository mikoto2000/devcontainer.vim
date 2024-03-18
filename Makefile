APP_NAME := devcontainer.vim

GO_BIN := ${GOPATH}/bin
LD_VLAGS := "-s -w"
VERSION := 0.0.1
OS := linux,windows,darwin
ARCH := amd64

DEST := ./build

### 開発関連
# 開発環境の都合で、個別にビルドできるようにしている
# (Linux コンテナ上でコーディングを行い、 Windows 上で実行することがあるため)
all: build
build: build/devcontainer.vim
build/devcontainer.vim:
	go build -ldflags="-s -w" -trimpath -o ./build/${APP_NAME}

build-all: build-windows build-linux build-darwin

build-windows: build/windows/devcontainer.vim.exe
build/windows/devcontainer.vim.exe:
	GOOS=windows GOARCH=amd64 go build -o build/windows/devcontainer.vim.exe ./main.go

build-linux: build/linux/devcontainer.vim
build/linux/devcontainer.vim:
	GOOS=linux GOARCH=amd64 go build -o build/linux/devcontainer.vim ./main.go

build-darwin: build/darwin/devcontainer.vim
build/darwin/devcontainer.vim:
	GOOS=darwin GOARCH=amd64 go build -o build/darwin/devcontainer.vim ./main.go

### リリース関連
# goxz を用いてリリース用のアーカイブを作成する
release-build: $(GO_BIN) main.go
	goxz -n ${APP_NAME} -o ${APP_NAME} -pv ${VERSION} -os=${OS} -arch=${ARCH} -d ${DEST}

${GO_BIN}/goxz:
	go install github.com/Songmu/goxz/cmd/goxz@latest

.PHONY: clean
clean:
	rm -rf build

