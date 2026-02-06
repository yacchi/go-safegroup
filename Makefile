.PHONY: help prepare test test-matrix test-go120 test-go121 test-go122 test-go123 test-go124 fmt

GOCACHE := $(CURDIR)/.gocache
GOPATH := $(CURDIR)/.gopath
GOMODCACHE := $(GOPATH)/pkg/mod
GOTMPDIR := $(CURDIR)/.gotmp
GOENV = env GOCACHE=$(GOCACHE) GOPATH=$(GOPATH) GOMODCACHE=$(GOMODCACHE) GOTMPDIR=$(GOTMPDIR)

help:
	@echo "Available targets:"
	@echo "  make test         - Run tests with current Go"
	@echo "  make test-matrix  - Run tests with Go 1.20, 1.21, 1.22, 1.23, 1.24 via mise"
	@echo "  make test-go120   - Run tests with Go 1.20 via mise"
	@echo "  make test-go121   - Run tests with Go 1.21 via mise"
	@echo "  make test-go122   - Run tests with Go 1.22 via mise"
	@echo "  make test-go123   - Run tests with Go 1.23 via mise"
	@echo "  make test-go124   - Run tests with Go 1.24 via mise"
	@echo "  make fmt          - Format Go files"

prepare:
	mkdir -p $(GOCACHE) $(GOMODCACHE) $(GOTMPDIR)

test: prepare
	$(GOENV) go test ./...

test-matrix: prepare
	$(GOENV) mise x go@1.20 -- go test ./...
	$(GOENV) mise x go@1.21 -- go test ./...
	$(GOENV) mise x go@1.22 -- go test ./...
	$(GOENV) mise x go@1.23 -- go test ./...
	$(GOENV) mise x go@1.24 -- go test ./...

test-go120: prepare
	$(GOENV) mise x go@1.20 -- go test ./...

test-go121: prepare
	$(GOENV) mise x go@1.21 -- go test ./...

test-go122: prepare
	$(GOENV) mise x go@1.22 -- go test ./...

test-go123: prepare
	$(GOENV) mise x go@1.23 -- go test ./...

test-go124: prepare
	$(GOENV) mise x go@1.24 -- go test ./...

fmt:
	gofmt -w *.go
