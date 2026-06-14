MODULE := github.com/flexphere/keyvind

# Library modules held to the full quality bar. The core (.) is dependency-free;
# each adapter under adapters/ is a nested module with its own go.mod.
# adapters/teakitv2 targets bubbletea v2 and requires Go 1.25+.
GO_MODULES := . adapters/teakit adapters/teakitv2 adapters/tcellkit

# Released modules: the root (tagged vX.Y.Z) plus each adapter (tagged
# adapters/<name>/vX.Y.Z). See RELEASING.md.
ADAPTER_MODULES := adapters/teakit adapters/teakitv2 adapters/tcellkit

# Runnable demos. Built/vetted but not held to the library lint bar.
EXAMPLE_MODULES := examples/bubbletea examples/tview

.DEFAULT_GOAL := all

.PHONY: all ci fmt fmt-check vet lint test test-race cover tidy tidy-check examples clean help tag tag-push

## all: format-check + vet + lint + test across library modules — local quality gate
all: fmt-check vet lint test

## ci: stricter gate used in CI (tidy-check + race + examples)
ci: fmt-check tidy-check vet lint test-race examples

## fmt: format every Go source in the repo
fmt:
	gofmt -w .
	goimports -w -local $(MODULE) .

## fmt-check: fail if any file is not formatted (covers all modules)
fmt-check:
	@bad="$$(gofmt -l .)"; \
	if [ -n "$$bad" ]; then \
		echo "gofmt needed on:"; echo "$$bad"; \
		echo "run 'make fmt'"; exit 1; \
	fi

## vet: go vet in each module
vet:
	@for m in $(GO_MODULES); do echo ">> vet $$m"; (cd $$m && go vet ./...) || exit 1; done

## lint: golangci-lint (v2) in each module
lint:
	@for m in $(GO_MODULES); do echo ">> lint $$m"; (cd $$m && golangci-lint run) || exit 1; done

## test: unit tests in each module
test:
	@for m in $(GO_MODULES); do echo ">> test $$m"; (cd $$m && go test ./...) || exit 1; done

## test-race: unit tests with the race detector in each module
test-race:
	@for m in $(GO_MODULES); do echo ">> test-race $$m"; (cd $$m && go test -race ./...) || exit 1; done

## cover: print total coverage per module
cover:
	@for m in $(GO_MODULES); do \
		echo ">> cover $$m"; \
		(cd $$m && go test -coverprofile=coverage.out ./... >/dev/null && go tool cover -func=coverage.out | tail -1) || exit 1; \
	done

## tidy: go mod tidy in each module
tidy:
	@for m in $(GO_MODULES); do echo ">> tidy $$m"; (cd $$m && go mod tidy) || exit 1; done

## tidy-check: fail if any go.mod/go.sum is not tidy
tidy-check:
	@for m in $(GO_MODULES); do \
		(cd $$m && cp go.mod go.mod.bak && { [ -f go.sum ] && cp go.sum go.sum.bak || true; } && \
		go mod tidy && \
		if ! diff -q go.mod go.mod.bak >/dev/null 2>&1; then \
			echo "$$m: go.mod not tidy; run 'make tidy'"; mv go.mod.bak go.mod; { [ -f go.sum.bak ] && mv go.sum.bak go.sum || true; }; exit 1; \
		fi; \
		rm -f go.mod.bak go.sum.bak) || exit 1; \
	done

## examples: build and vet each runnable demo module (no binary artifacts)
examples:
	@for m in $(EXAMPLE_MODULES); do echo ">> build $$m"; (cd $$m && go build -o /dev/null ./... && go vet ./...) || exit 1; done

## tag: create semver tags for every released module (root + adapters). Usage: make tag VERSION=vX.Y.Z
tag:
	@test -n "$(VERSION)" || { echo "usage: make tag VERSION=vX.Y.Z"; exit 1; }
	@echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$$' || { echo "VERSION must be semver, e.g. v0.0.2"; exit 1; }
	@git diff --quiet || { echo "working tree is dirty; commit before tagging"; exit 1; }
	git tag $(VERSION)
	@for m in $(ADAPTER_MODULES); do git tag $$m/$(VERSION); done
	@echo ">> tagged $(VERSION) (root + $(ADAPTER_MODULES)). push with: make tag-push VERSION=$(VERSION)"

## tag-push: push the tags created by 'make tag' to origin
tag-push:
	@test -n "$(VERSION)" || { echo "usage: make tag-push VERSION=vX.Y.Z"; exit 1; }
	git push origin $(VERSION)
	@for m in $(ADAPTER_MODULES); do git push origin $$m/$(VERSION); done

## clean: remove build/test artifacts
clean:
	@for m in $(GO_MODULES); do rm -f $$m/coverage.out; done

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
