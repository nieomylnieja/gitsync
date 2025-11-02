.DEFAULT_GOAL := help
MAKEFLAGS += --silent --no-print-directory

BIN_DIR := ./bin
SCRIPTS_DIR := ./scripts
APP_NAME := gitsync
LDFLAGS += -s -w

# Print Makefile target step description for check.
# Only print 'check' steps this way, and not dependent steps, like 'install'.
# ${1} - step description
define _print_step
	printf -- '------\n%s...\n' "${1}"
endef

## Activate developer environment using devbox. Run `make install/devbox` first If you don't have devbox installed.
activate:
	devbox shell

## Install devbox binary.
install/devbox:
	$(call _print_step,Installing devbox)
	curl -fsSL https://get.jetpack.io/devbox | bash

## Update devbox manged package versions.
update/devbox:
	$(call _print_step,Update packages managed by devbox)
	devbox update

.PHONY: build
## Build gitsync binary.
build:
	$(call _print_step,Building binary)
	mkdir -p $(BIN_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

.PHONY: release
## Build and release the binaries.
release:
	$(call _print_step,Releasing binary)
	goreleaser release --snapshot --clean

.PHONY: test
## Run all unit tests.
test:
	$(call _print_step,Running unit tests)
	go test -race -cover ./...

.PHONY: test/coverage
## Produce test coverage report and inspect it in browser.
test/coverage:
	$(call _print_step,Running test coverage report)
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: check check/vet check/lint check/gosec check/spell check/trailing check/markdown check/generate check/vulnerabilities
## Run all checks.
check: check/vet check/lint check/gosec check/spell check/trailing check/markdown check/generate check/vulnerabilities

## Run 'go vet' on the whole project.
check/vet:
	$(call _print_step,Running go vet)
	go vet ./...

## Run golangci-lint all-in-one linter with configuration defined inside .golangci.yml.
check/lint:
	$(call _print_step,Running golangci-lint)
	golangci-lint run

## Check for security problems using gosec, which inspects the Go code by scanning the AST.
check/gosec:
	$(call _print_step,Running gosec)
	gosec -exclude-generated -quiet ./...

## Check spelling, rules are defined in cspell.json.
check/spell:
	$(call _print_step,Verifying spelling)
	cspell --no-progress '**/**'

## Check for trailing whitespaces in any of the projects' files.
check/trailing:
	$(call _print_step,Looking for trailing whitespaces)
	$(SCRIPTS_DIR)/check-trailing-whitespaces.bash

## Check markdown files for potential issues with markdownlint.
check/markdown:
	$(call _print_step,Verifying Markdown files)
	markdownlint '**/*.md' --ignore 'node_modules'

## Check for potential vulnerabilities across all Go dependencies.
check/vulnerabilities:
	$(call _print_step,Running govulncheck)
	govulncheck ./...

## Verify if the auto generated code has been committed.
check/generate:
	$(call _print_step,Checking if generated code matches the provided definitions)
	$(SCRIPTS_DIR)/check-generate.bash

.PHONY: generate generate/code
## Auto generate files.
generate: generate/code

## Generate Golang code.
generate/code:
	$(call _print_step,Generating Golang code...)
	go generate ./...

.PHONY: format format/go
## Format files.
format: format/go

## Format Go files.
format/go:
	$(call _print_step,Formatting Go files)
	golangci-lint fmt

.PHONY: help
## Print this help message.
help:
	$(SCRIPTS_DIR)/makefile-help.awk $(MAKEFILE_LIST)
