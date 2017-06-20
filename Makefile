###############################################################################
# set some flags to control Make's behavior
###############################################################################

# disable all builtin suffix pattern rules
.SUFFIXES:

# these rules should all run in parallel just fine
MAKEFLAGS := "-j "

# run all the rule commands in Bash with "-o pipefail" so failing commands with
# piped output still exit with the right status
SHELL = /bin/bash -o pipefail

# define all our top level targets
TARGETS := build build-release test fmt lint clean update-deps
.PHONY: $(TARGETS)

# the default will be to build for the current host platform/arch (for easy development)
.DEFAULT_GOAL := build


###############################################################################
# define some versioning logic (see "Versioning" in the README for details)
###############################################################################

# full Git commit hash of the revision we're building
SUBSTRATE_COMMIT := $(shell git rev-parse HEAD)

# append -dirty to the commit hash if the working directory has been modified
SUBSTRATE_COMMIT := $(SUBSTRATE_COMMIT)$(shell git status | grep -q 'added to commit' &&  echo -dirty)

# current version as tagged in ./VERSION
SUBSTRATE_VERSION := $(shell cat VERSION)

# if the current branch/tag == VERSION, we must be doing a final release build,
# otherwise we're doing a "snapshot" build.
ifneq ($(SUBSTRATE_VERSION),$(shell git describe --tags))
SUBSTRATE_VERSION := $(SUBSTRATE_VERSION)-snapshot
endif


###############################################################################
# set up some cross-compilation variables
###############################################################################

# detect the host OS/arch (where this make is running)
export HOST_TARGET ?= $(shell uname -s | tr A-Z a-z)-$(subst x86_64,amd64,$(shell uname -m))

# target platform/architectures for when we want to build a full release
export RELEASE_TARGETS ?= linux-amd64 darwin-amd64

# define some helpers to extract just the platform or architecture from our target,
# assuming the target is the pattern stem (%) value
export PLATFORM = $(word 1, $(subst -, ,$*))
export ARCH = $(word 2, $(subst -, ,$*))


###############################################################################
# define some additional top level variables
###############################################################################

# what version of Terraform we're bundling
export TERRAFORM_VERSION := 0.8.3

# this is a directory we'll cache downloaded files in by default, we
# can override this to point at $HOME/... in Jenkins to keep cached files
# between builds
export CACHE_DIR ?= $(CURDIR)/cache

# this is the directory we'll use to store all intermediate build products
export BUILD_DIR := $(CURDIR)/build

# this is the directory we'll use for the final output binaries
export BIN_DIR := $(CURDIR)/bin

# we're going to do all builds inside an isolated GOPATH under BUILD_DIR
export GOPATH := $(BUILD_DIR)/gopath

# define a separate GOPATH only for tools that we need during the build
export TOOLS_GOPATH := $(CACHE_DIR)/tools-$(HOST_TARGET)


###############################################################################
# define top level targets (ones you might type on the command line)
###############################################################################

# build (default target) compiles substrate for the current host platform
# and copy to ./bin/substrate for convenience
build: bin/substrate
bin/substrate: $(BIN_DIR)/substrate-$(HOST_TARGET)-$(SUBSTRATE_VERSION)
	@cp -v $< $@ && touch $@

# build-release compiles substrate for all supported target platforms
build-release: $(patsubst %,$(BIN_DIR)/substrate-%-$(SUBSTRATE_VERSION),$(RELEASE_TARGETS))

# clean should essentially revert back to a clean checkout
# it should always run before any other top level targs
clean:
	rm -rf $(BIN_DIR) $(BUILD_DIR)

# fmt and lint are broken out into their own Makefile
fmt lint: $(BUILD_DIR)/$(HOST_TARGET)/terraform
	@$(MAKE) -C util $@ TERRAFORM=$< SOURCE_DIR=$(CURDIR) GOPATH=$(TOOLS_GOPATH) | sed -e 's/^/[$@] /'

# test only runs linters and checks that things compile for now
test: lint build-release
	@echo WARNING: Substrate only has lint checks for now, not any real tests!


###############################################################################
# define rules to set up the Go build environment (isolated GOPATH)
###############################################################################

# directory of the substrate directory within GOPATH
SUBSTRATE_PKG_DIR := $(GOPATH)/src/github.com/SimpleFinance/substrate

# copied_go_sources returns a list of all the the .go source files, translated
# into their destination path under the GOPATH
copied_go_sources = $(patsubst %, $(SUBSTRATE_PKG_DIR)/%, $(shell find $1 -type f -name '*.go'))
# copy our source files into the right place within the GOPATH package directory on demand
$(SUBSTRATE_PKG_DIR)/%: %
	@mkdir -p $(@D)
	@cp $< $@
.PRECIOUS: $(SUBSTRATE_PKG_DIR)/%

# GO_DEPS is a marker file so we can set up the Go environment when we need it
# it needs to be updated whenever our Glide-specified dependencies change
GO_DEPS := $(SUBSTRATE_PKG_DIR)/vendor/installed
$(GO_DEPS): glide.yaml glide.lock
	@(go version | grep -q "go version") || (echo "You don't seem to have a valid Go toolchain, maybe GOROOT isn't set?"; exit 1)
	@mkdir -p $(@D)
	@cp glide.yaml glide.lock $(SUBSTRATE_PKG_DIR)
	cd $(SUBSTRATE_PKG_DIR) && glide --no-color install 2>&1 | sed -e 's/^/[glide] /'
	@touch $@

# update-deps updates the glide.lock file with the latest updates to glide.yaml
# and any new package versions (subject to version constraints in glide.yaml)
update-deps: $(GO_DEPS)
	cd $(SUBSTRATE_PKG_DIR) && glide --no-color update
	cp $(SUBSTRATE_PKG_DIR)/glide.lock glide.lock


###############################################################################
# install tools we need to do the build itself
###############################################################################

# go-bindata is a tool for bundling binary assets into a .go source file
GOBINDATA := $(TOOLS_GOPATH)/bin/go-bindata
$(GOBINDATA):
	GOPATH=$(TOOLS_GOPATH) go get -u github.com/jteeuwen/go-bindata/...
	@touch $@


###############################################################################
# define rules to build our custom Terraform provider plugins
###############################################################################

# our custom terraform providers
CUSTOM_PROVIDERS := \
	terraform-provider-bakery \
	terraform-provider-tarball

# messy macro-laden loop which defines the build rules for all the custom providers
CUSTOM_TERRAFORM_BINARIES:=
define build_provider
CUSTOM_TERRAFORM_BINARIES += $(BUILD_DIR)/%/$(1)
$$(BUILD_DIR)/%/$(1): $$(GO_DEPS) $$(call copied_go_sources, providers/$(1))
	GOOS=$$(PLATFORM) GOARCH=$$(ARCH) go build -o $$@ github.com/SimpleFinance/substrate/providers/$(1)
	@touch $$@
endef
$(foreach p, $(CUSTOM_PROVIDERS),$(eval $(call build_provider,$(p))))
.PRECIOUS: $(CUSTOM_TERRAFORM_BINARIES)


###############################################################################
# define rules to download and extract the builtin Terraform providers we need
###############################################################################

#the builtin terraform binaries on which we depend
BUILTIN_PROVIDERS := terraform

# download the Terraform release zip for a particular platform/arch
CACHED_TERRAFORM_ZIP := $(CACHE_DIR)/terraform-$(TERRAFORM_VERSION)-%.zip
.PRECIOUS: $(CACHED_TERRAFORM_ZIP) # don't delete the cached zip between builds
$(CACHED_TERRAFORM_ZIP):
	@mkdir -p $(@D)
	curl -s -o $@ https://releases.hashicorp.com/terraform/$(TERRAFORM_VERSION)/terraform_$(TERRAFORM_VERSION)_$(PLATFORM)_$(ARCH).zip
	@touch $@

TERRAFORM_ZIP := $(BUILD_DIR)/%/terraform-$(TERRAFORM_VERSION).zip
.PRECIOUS: $(TERRAFORM_ZIP)
$(TERRAFORM_ZIP): $(CACHED_TERRAFORM_ZIP)
	@mkdir -p $(@D)
	@cp $< $@

# extract the builtin terraform binaries we want into $BUILD_DIR/$TARGET/
BUILTIN_TERRAFORM_BINARIES := $(BUILD_DIR)/%/terraform

$(BUILD_DIR)/%/terraform: $(TERRAFORM_ZIP)
	@mkdir -p $(@D)
	unzip -p $< $(notdir $@) > $@
	@chmod +x $@ && touch $@
.PRECIOUS: $(BUILTIN_TERRAFORM_BINARIES)

###############################################################################
# define rules to bundle the (platform/arch specifc) binaries into a .go file
###############################################################################

# define a rule to bundle all our custom and builtin binaries into a .go source file
CLI_BUNDLE_BINARIES := $(SUBSTRATE_PKG_DIR)/cmd/substrate/assets/binaries/data-%.go
$(CLI_BUNDLE_BINARIES): $(CUSTOM_TERRAFORM_BINARIES) $(BUILTIN_TERRAFORM_BINARIES) | $(GOBINDATA)
	$(GOBINDATA) \
		-pkg binaries \
		-nomemcopy \
		-nocompress \
		-tags "$(PLATFORM),$(ARCH)" \
		-prefix $(BUILD_DIR)/$(PLATFORM)-$(ARCH)/ \
		-o $@ \
		$^
	@touch $@
.PRECIOUS: $(CLI_BUNDLE_BINARIES)


###############################################################################
# define rules to bundle the zone configuration (./zone) into a .go file
###############################################################################

# generate a bundle of the ./zone directory containing all our configuration files
CLI_BUNDLE_ZONE_CONFIG := $(SUBSTRATE_PKG_DIR)/cmd/substrate/assets/zoneconfig/data.go
$(CLI_BUNDLE_ZONE_CONFIG): $(shell find ./zone -not -iname "*~" -type f) | $(GOBINDATA)
	$(GOBINDATA) \
		-pkg zoneconfig \
		-nomemcopy \
		-nocompress \
		-o $@ \
		$^
	@touch $@
.PRECIOUS: $(CLI_BUNDLE_ZONE_CONFIG)


###############################################################################
# define the rule for building the final CLI binary from everything above
###############################################################################

# build the final `substrate` CLI binary for a particular target
$(BIN_DIR)/substrate-%-$(SUBSTRATE_VERSION): $(CLI_BUNDLE_BINARIES) $(CLI_BUNDLE_ZONE_CONFIG) $(call copied_go_sources, cmd)
	@mkdir -p $(@D)
	GOOS=$(PLATFORM) GOARCH=$(ARCH) go build \
		 -o $@ \
		 -tags "$(PLATFORM),$(ARCH)" \
		 -ldflags "-X main.version=$(SUBSTRATE_VERSION) -X main.commit=$(SUBSTRATE_COMMIT)" \
		 github.com/SimpleFinance/substrate/cmd/substrate
	@touch $@
