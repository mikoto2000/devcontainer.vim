all: build-all

build-all: build-windows build-linux build-darwin

build-windows: main.go
	GOOS=windows GOARCH=amd64 go build -o build/windows/dcvim.exe ./main.go

build-linux: main.go
	GOOS=linux GOARCH=amd64 go build -o build/linux/dcvim ./main.go

build-darwin: main.go
	GOOS=darwin GOARCH=amd64 go build -o build/darwin/dcvim ./main.go

.PHONY: clean
clean:
	rm -rf build
