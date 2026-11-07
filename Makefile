APP_NAME := devcontainer.vim

PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64 \
	windows/arm64

GO_BIN := ${GOPATH}/bin
VERSION := 4.0.1
LD_FLAGS := "-s -w -X main.version=${VERSION}"

DEST := ./build

### 開発関連
# 開発環境の都合で、個別にビルドできるようにしている
# (Linux コンテナ上でコーディングを行い、 Windows 上で実行することがあるため)
.PHONY: all build force-build
all: build
build: build/devcontainer.vim
# Let Go track the full set of build inputs and reuse cached packages.
build/devcontainer.vim: force-build
	go build -ldflags=${LD_FLAGS} -trimpath -o ./build/${APP_NAME}

force-build:

build-all:
	@mkdir -p $(DEST)
	@set -e; \
	for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; GOARCH=$${platform#*/}; \
		ext=""; [ $$GOOS = "windows" ] && ext=".exe"; \
		out="$(DEST)/$(APP_NAME)-$${GOOS}-$${GOARCH}$$ext"; \
		echo "Building $$out"; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags "-s -w -X main.version=$(VERSION)" -o $$out .; \
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

