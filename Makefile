
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

## build/debug: build with specific flags that allows delve debugging on remote port
.PHONY: build/debug
build/debug:
	CGOENABLED=1; \
	go build -gcflags "all=-N -l" -o ./bin ./cmd/letshare; \
	dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./bin/letshare.exe