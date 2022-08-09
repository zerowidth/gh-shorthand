APP    = bin/gh-shorthand
GOSRC  = $(shell find . -type f -name '*.go')

# V=1 for verbose
V = 0
Q = $(if $(filter 1,$V),,@)

default: build
all: build test lint

$(APP): $(GOSRC) go.mod go.sum; $(info -> building gh-shorthand...)
	$Q go build -o $(APP) .

build: $(APP)

lint: | $(GOLINT); $(info -> running linters...)
	$Q go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.48.0 run

TESTSUITE = ./...
TESTFLAGS =

.PHONY: test
test: | ; $(info -> running tests...)
	$Q go run github.com/rakyll/gotest@latest $(TESTFLAGS) $(TESTSUITE)

.PHONY: clean
clean:
	$Q rm -rf $(APP)
	$Q go clean -testcache ./...
