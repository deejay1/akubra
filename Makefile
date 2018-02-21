VERSION := `cat VERSION`
LDFLAGS := -X main.version=$(VERSION)
GO := "$(GOROOT)/bin/go"

all: formatting lint test build

formatting :
	$(GO) get golang.org/x/tools/cmd/goimports

lint: deps-lint
	gometalinter ./... \
	--disable=gotype \
	--disable=dupl \
	--deadline=600s \
	--disable=goimports \
	--vendor \
	--fast

lint-slow: deps-lint
	gometalinter ./... \
	--disable=gotype \
	--disable=dupl \
	--deadline=600s \
	--enable=goimports \
	--vendor 


deps:
	$(GO) get github.com/Masterminds/glide
	glide install

deps-lint: deps
	$(GO) get github.com/alecthomas/gometalinter
	gometalinter --install

build: deps lint
        # Enable netcgo, then name resolution will use systems dns caches
	$(GO) build -v -ldflags "$(LDFLAGS)" -tags 'netcgo=1'.

test: deps
	$(GO) test -v -race -cover $$(go list ./... | grep -v /vendor/)

clean:
	$(GO) clean .
