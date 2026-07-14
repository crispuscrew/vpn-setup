# vpn-setup - build the Go tooling and run the pipeline in a container.
# `make help` (default) lists targets. The Go build/test/lint run inside a pinned
# golang image (dev-rules/tech/containers.md); nothing touches the host toolchain.

.DEFAULT_GOAL := help

# Prefer rootless Podman, fall back to Docker - never hardcode one.
ENGINE := $(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)
GO_IMAGE := docker.io/library/golang:1.24.10-alpine
VPNBOT_IMAGE ?= vpn-setup/vpnbot:dev

# Run a go command in the pinned container, repo mounted at /src. CGO off so the
# binaries are static and run on the host (musl-built image, glibc host).
GORUN = $(ENGINE) run --rm -v "$(CURDIR)":/src:Z -w /src \
	-e CGO_ENABLED=0 -e GOFLAGS=-trimpath -e GOCACHE=/src/.cache/go-build \
	$(GO_IMAGE)

help:  ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'

build:  ## Build both binaries (bin/vpn, bin/vpnbot) in the container
	$(GORUN) go build -ldflags '-s -w' -o bin/vpn ./cmd/vpn
	$(GORUN) go build -ldflags '-s -w' -o bin/vpnbot ./cmd/vpnbot

run: build  ## Build, then run `vpn help`
	./bin/vpn help

test:  ## Run go tests in the container
	$(GORUN) go test ./...

vet:  ## Run go vet in the container
	$(GORUN) go vet ./...

fmt:  ## Fail if any first-party Go file is not gofmt-formatted (vendor excluded)
	$(GORUN) sh -c 'files=$$(find . \( -path ./vendor -o -path ./.cache \) -prune -o -name "*.go" -print); unformatted=$$(gofmt -l $$files); test -z "$$unformatted" || { echo "$$unformatted"; exit 1; }'

lint: vet fmt  ## go vet + gofmt check

image:  ## Build the vpnbot container image (VPNBOT_IMAGE to override the tag)
	$(ENGINE) build -f cmd/vpnbot/Containerfile -t $(VPNBOT_IMAGE) .

clean:  ## Remove build output and the go cache
	rm -rf bin .cache

.PHONY: help build run test vet fmt lint image clean
