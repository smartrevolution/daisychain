GO   := go
pkgs  = $(shell $(GO) list ./...)

all: format build test

test:
	@echo ">> running tests"
	@$(GO) test $(pkgs) -cover

bench:
	@echo ">> running benchmarks"
	@$(GO) test $(pkgs) -bench=-

vtest:
	@echo ">> running verbose tests"
	@$(GO) test $(pkgs) -v -chatty

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

build:
	@echo ">> building binaries"
	@$(GO) build $(pkgs)

.PHONY: all test bench vtest format vet build 
