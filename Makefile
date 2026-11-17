BIN     := braise
VERSION ?= 0.1.0
LDFLAGS := -s -w -X main.version=$(VERSION)
OUT     := bin

.PHONY: all build run vet test clean dist windows macos linux checksums sign-windows sign-macos

all: build

build:
	mkdir -p $(OUT)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT)/$(BIN) .

run:
	go run . --dir ./Braise

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf $(OUT)

windows:
	mkdir -p $(OUT)
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT)/$(BIN)-windows-amd64.exe .

macos:
	mkdir -p $(OUT)
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT)/$(BIN)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT)/$(BIN)-darwin-amd64 .

linux:
	mkdir -p $(OUT)
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT)/$(BIN)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT)/$(BIN)-linux-arm64 .

dist: windows macos linux

checksums:
	./scripts/checksums.sh bin

sign-windows:
	pwsh ./scripts/sign-windows.ps1 -Path bin/braise-windows-amd64.exe

sign-macos:
	./scripts/sign-macos.sh bin/braise-darwin-arm64 $(VERSION)
