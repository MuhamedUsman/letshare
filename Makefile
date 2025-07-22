include .envrc

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## dev/debug: build with specific flags that allows delve debugging on remote port (GOLAND specific)
.PHONY: dev/debug
dev/debug:
	CGO_ENABLED=1; \
	go build -gcflags "all=-N -l" -o ./bin .; \
	dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./bin/letshare.exe

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build/app: build the application binary
.PHONY: build/app
build/app:
	CGO_ENABLED=0; \
	go build -ldflags="-s -w -X main.version=${VERSION}" .

## goreleaser/snapshot: build the goreleaser test dist
.PHONY: goreleaser/snapshot
goreleaser/snapshot:
	goreleaser release --snapshot --clean --skip=publish

## goreleaser/release: release it to the world
.PHONY: goreleaser/release
goreleaser/release:
	goreleaser release --clean --skip=publish