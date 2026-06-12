SHELL := /bin/bash

BIN      := rkmon
PKG      := github.com/isac322/rkmon
VERSION  := 0.2.0
GIT_SHA  := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
LDFLAGS  := -s -w \
            -X 'main.Version=$(VERSION)' \
            -X 'main.GitSHA=$(GIT_SHA)'

NAS      ?= your-rk3588-host
INSTALL_DIR_USER ?= $(HOME)/.local/bin
INSTALL_DIR_SYS  ?= /usr/local/bin

.PHONY: all
all: build

.PHONY: build
build:
	@mkdir -p build
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o build/$(BIN)-linux-arm64 ./cmd/rkmon
	@ls -la build/$(BIN)-linux-arm64

.PHONY: build-host
build-host:
	@mkdir -p build
	@go build -trimpath -ldflags "$(LDFLAGS)" -o build/$(BIN) ./cmd/rkmon

.PHONY: test
test:
	@go test ./...

.PHONY: vet
vet:
	@go vet ./...

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: deploy
deploy: build
	@echo "==> scp build/$(BIN)-linux-arm64 -> $(NAS):/tmp/$(BIN)"
	@scp -O build/$(BIN)-linux-arm64 $(NAS):/tmp/$(BIN)
	@echo "==> install to $(INSTALL_DIR_USER)/$(BIN) on $(NAS)"
	@ssh $(NAS) "install -m 0755 /tmp/$(BIN) $(INSTALL_DIR_USER)/$(BIN) && rm -f /tmp/$(BIN) && $(INSTALL_DIR_USER)/$(BIN) --version"

.PHONY: deploy-sys
deploy-sys: build
	@echo "==> scp build/$(BIN)-linux-arm64 -> $(NAS):/tmp/$(BIN)"
	@scp -O build/$(BIN)-linux-arm64 $(NAS):/tmp/$(BIN)
	@echo "==> install (sudo) to $(INSTALL_DIR_SYS)/$(BIN) on $(NAS)"
	@ssh -t $(NAS) "sudo install -m 0755 -o root -g root /tmp/$(BIN) $(INSTALL_DIR_SYS)/$(BIN) && rm -f /tmp/$(BIN) && $(INSTALL_DIR_SYS)/$(BIN) --version"

.PHONY: clean
clean:
	@rm -rf build/
