APP_NAME := devcontainer.vim

PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64 \
	windows/arm64

GO_BIN := ${GOPATH}/bin
VERSION := 4.0.0-beta2
LD_FLAGS := "-s -w -X main.version=${VERSION}"

DEST := ./build

WATCH_SRC := ./main.go \
						 ./devcontainer/DevcontainerJson.go \
						 ./devcontainer/devcontainer.go \
						 ./devcontainer/run.go \
						 ./devcontainer/start.go \
						 ./devcontainer/dockerRunVimArgs.go \
						 ./devcontainer/devcontainerStartVimArgs.go \
						 ./devcontainer/readConfigurationResult.go \
						 ./devcontainer/upCommandResult.go \
						 ./devcontainer/VimRun_aarch64.template.sh \
						 ./devcontainer/VimRun_system.template.sh \
						 ./devcontainer/VimRun_x86_64_AppImage.template.sh \
						 ./devcontainer/VimRun_x86_64_static.template.sh \
						 ./docker/docker.go \
						 ./docker/dockerPsResult.go \
						 ./dockercompose/dockerCompose.go \
						 ./dockercompose/dockerComposePsResult.go \
						 ./tools/tools.go \
						 ./tools/vim.go \
						 ./tools/nvim.go \
						 ./tools/devcontainer.go \
						 ./tools/devcontainer_darwin.go \
						 ./tools/devcontainer_linux.go \
						 ./tools/devcontainer_windows.go \
						 ./tools/clipboard-data-receiver.go \
						 ./tools/port-forwarder.go \
						 ./util/port-forwarder.go \
						 ./util/util.go

### 開発関連
# 開発環境の都合で、個別にビルドできるようにしている
# (Linux コンテナ上でコーディングを行い、 Windows 上で実行することがあるため)
all: build
build: build/devcontainer.vim
build/devcontainer.vim: ${WATCH_SRC}
	go build -ldflags=${LD_FLAGS} -trimpath -o ./build/${APP_NAME}

build-all:
	@mkdir -p $(DEST)
	@set -e; \
	for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; GOARCH=$${platform#*/}; \
		ext=""; [ $$GOOS = "windows" ] && ext=".exe"; \
		out="$(DEST)/$(APP_NAME)-$${GOOS}-$${GOARCH}$$ext"; \
		echo "Building $$out"; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags "-s -w -X main.version=$(VERSION)" -o $$out $(PKG); \
	done


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


### テスト関連
.PHONY: test
test:
	go test -cover ./... -coverprofile=cover.out
	go tool cover -html=cover.out -o cover.html

