TOOLS  = _tools/bin
APP    = bin/gh-shorthand
GOSRC  = $(shell find . -type f -name '*.go')
GOLINT = $(TOOLS)/golangci-lint
GOTEST = $(TOOLS)/gotest

# V=1 for verbose
V = 0
Q = $(if $(filter 1,$V),,@)

default: build
all: build test lint

$(APP): $(GOSRC) go.mod go.sum; $(info -> building gh-shorthand...)
	$Q go build -o $(APP) .

build: $(APP)

lint: | $(GOLINT); $(info -> running linters...)
	$Q $(GOLINT) run \
		--enable goimports \
		--enable unparam \
		--enable dupl \
		--enable interfacer

$(GOLINT): $(TOOLS)
$(GOTEST): $(TOOLS)

$(TOOLS): ; $(info -> installing tools...)
	$Q script/bootstrap

TESTSUITE = ./...
.PHONY: test
test: | $(GOTEST); $(info -> running tests...)
	$Q $(GOTEST) $(TESTFLAGS) $(TESTSUITE)

.PHONY: clean
clean:
	$Q rm -rf $(APP) $(TOOLS)
	$Q go clean -testcache ./...
