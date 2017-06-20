# this Makefile handles autoformatting (`make fmt`) and linting (`make lint`).
# It expects to be run from the top level Makefile with variables
# BUILD_DIR/SOURCE_DIR/TERRAFORM/HOST_TARGET set.

# these tools should all run in parallel
MAKEFLAGS := "-j "

# all of these targets are phony, even though they actually exist as scripts on disk
TARGETS := fmt fmt-go fmt-json fmt-sh fmt-style fmt-tf lint lint-go lint-json lint-sh lint-style lint-tf lint-yaml
.PHONY: $(TARGETS)

# we shouldn't need to install gofmt, we just need to find it or error out if it's missing
export GOFMT := $(shell which gofmt || echo missing-gofmt)
$(GOFMT):
	$(error "could not find 'gofmt' on your PATH")

export GOLINT := $(GOPATH)/bin/golint
$(GOLINT):
	go get -u github.com/golang/lint/golint
	@touch $@

export JQ := $(GOPATH)/bin/jq
$(JQ):
	@mkdir -p $(@D)
ifeq ($(HOST_TARGET),darwin-amd64)
	curl -sLo $@ https://github.com/stedolan/jq/releases/download/jq-1.5/jq-osx-amd64
else ifeq ($(HOST_TARGET),linux-amd64)
	curl -sLo $@ https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64
else
	$(error don\'t know how to download jq on $(HOST_TARGET))
endif
	@chmod +x $@
	@touch $@

# we don't have a builtin installed for shellcheck yet, just error out if it's not available
export SHELLCHECK := $(shell which shellcheck || echo missing-shellcheck)
$(SHELLCHECK):
	$(error "could not find 'shellcheck' on your PATH (see https://github.com/koalaman/shellcheck)")

export SHFMT := $(GOPATH)/bin/shfmt
$(SHFMT):
	go get -u github.com/mvdan/sh/cmd/shfmt
	@touch $@

export YAML := $(GOPATH)/bin/yaml
$(YAML):
	go get -u github.com/mikefarah/yaml
	@touch $@


# define the dependencies for some of the individual fmt/lint tools
fmt-go lint-go: $(GOFMT) $(GOLINT)
fmt-sh lint-sh: $(SHFMT) $(SHELLCHECK)
fmt-json lint-json: $(JQ)
lint-yaml: $(YAML)
fmt-tf lint-tf: $(TERRAFORM)

# all the fmt/lint targets run their respective ./util/* scripts
fmt-go fmt-json fmt-sh fmt-style fmt-tf lint-go lint-json lint-sh lint-style lint-tf lint-yaml:
	@echo running $@...
	@$(CURDIR)/$@

# the top level fmt and lint targets run all the respective subtargets
fmt: fmt-go fmt-json fmt-sh fmt-style fmt-tf
lint: lint-go lint-json lint-sh lint-style lint-tf lint-yaml
