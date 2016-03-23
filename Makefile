VERSION := 0.0.1

LDFLAGS := -X main.Version=$(VERSION)
GOFLAGS := -ldflags "$(LDFLAGS)"
GOOS ?= $(shell uname | tr A-Z a-z)
GOARCH ?= $(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
SUFFIX ?= $(GOOS)-$(GOARCH)
ARCHIVE ?= $(BINARY)-$(VERSION).$(SUFFIX).tar.gz
BINARY := tyson-$(VERSION).$(SUFFIX)

./dist/$(BINARY):
	mkdir -p ./dist
	go build $(GOFLAGS) -o $@

.PHONY: test
test:
	go test $$(go list ./... | grep -v /vendor/)

.PHONY: clean
clean:
	rm -rf ./dist

.PHONY: docker
docker:
	docker run --rm -v "$$PWD":/go/src/github.com/iamseth/tyson -w /go/src/github.com/iamseth/tyson golang:1.6 bash -c make
