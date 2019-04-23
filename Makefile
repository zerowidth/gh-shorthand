TOOLS  = _tools/bin
APP    = bin/gh-shorthand
GOSRC  = $(shell find . -type f -name '*.go')
GOLINT = $(TOOLS)/golangci-lint

# V=1 for verbose
V = 0
Q = $(if $(filter 1,$V),,@)

default: build
all: build test lint

$(GOPATH): ; $(info setting GOPATH...)
	$Q mkdir -p $@

$(APP): $(GOSRC); $(info building gh-shorthand...)
	$Q go build -o $(APP) ./cmd

build: $(APP)

lint: | $(GOLINT); $(info running linters...)
	$Q $(GOLINT) run \
		--enable goimports \
		--enable unparam \
		--enable dupl \
		--enable interfacer

$(GOLINT): $(TOOLS)

$(TOOLS): ; $(info installing tools...)
	$Q script/bootstrap

TESTFLAGS = -race
TESTSUITE = ./...
.PHONY: test
test: ; $(info running tests...)
	$Q go test $(TESTFLAGS) $(TESTSUITE)

.PHONY: clean
clean:
	$Q rm -rf $(APP) $(TOOLS)
