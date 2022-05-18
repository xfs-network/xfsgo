
PWD := $(shell pwd)

PREFIX ?= /usr/local
BINDIR ?= ${PREFIX}/bin

GOPATH := $(shell go env GOPATH)

APP=xfsgo

all: ${APP}

${APP}:
	@go build -o $(PWD)/$@ ./cmd/$@
	@echo "Build successful"

devtools:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0 1>/dev/null

precheck: devtools
	@${GOPATH}/bin/golangci-lint run 1>/dev/null

test: precheck
	@go test ./...
	@echo "Test successful"

install:
	@cp ${PWD}/${APP} ${BINDIR}/${APP}
	@chmod 755 ${BINDIR}/${APP}
	@echo "Installation succeeded"

clean:
	@go clean -cache
	@rm -f ${PWD}/${APP}
	@echo "Already clear build cache files"

.PHONY: devtools precheck test install clean
.PHONY: ${APP}