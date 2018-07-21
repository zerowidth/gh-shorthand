GOPATH = $(CURDIR)/.gopath
GOBIN  = $(GOPATH)/bin
BASE   = $(GOPATH)/src/github.com/zerowidth/gh-shorthand
APP    = $(BASE)/bin/gh-shorthand
GOSRC  = $(shell find . -type f -name '*.go' | grep -v .gopath)
GODEP  = $(GOBIN)/dep
GOLINT = $(GOBIN)/golangci-lint

V = 0
Q = $(if $(filter 1,$V),,@)

default: build test lint

$(GOPATH): ; $(info setting GOPATH...)
	$Q mkdir -p $@

$(BASE): | $(GOPATH)
	$Q mkdir -p $(dir $@)
	$Q ln -s $(CURDIR) $@

$(APP): $(GOSRC) | $(BASE)
	$Q cd $(BASE) && GOPATH=$(GOPATH) go build -o $(APP) ./cmd

build: $(APP); $(info building gh-shorthand...)

lint: | $(GOLINT) $(BASE); $(info running linters...)
	$Q cd $(BASE) && GOPATH=$(GOPATH) $(GOLINT) run \
		--enable goimports \
		--enable unparam \
		--enable dupl \
		--enable interfacer

TESTFLAGS = -race
TESTSUITE = ./...
.PHONY: test
test: | $(BASE); $(info running tests...)
	$Q cd $(BASE) && GOPATH=$(GOPATH) go test $(TESTFLAGS) $(TESTSUITE)

$(GOLINT): | $(GOPATH); $(info installing golangci-lint...)
	$Q GOPATH=$(GOPATH) go get github.com/golangci/golangci-lint/cmd/golangci-lint

$(GODEP): | $(GOPATH); $(info installing dep...)
	$Q GOPATH=$(GOPATH) go get github.com/golang/dep/cmd/dep
.PHONY: dep
dep: | $(GODEP)

.PHONY: clean
clean:
	$Q rm -rf $(APP) $(GOPATH)
