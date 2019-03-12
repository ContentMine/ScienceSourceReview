BASIC=Makefile
GO=go
GIT=git

$(eval VERSION:=$(shell git rev-parse HEAD)$(shell git diff --quiet || echo '*'))
$(eval REMOTE:=$(shell git remote get-url origin))
LD_FLAGS=-ldflags "-X main.Version=${VERSION} -X main.Remote=${REMOTE}"

all: .PHONY ScienceSourceIngest

ScienceSourceIngest: .PHONY check-env
	$(GO) install $(LD_FLAGS) github.com/ContentMine/ScienceSourceReview

fmt: .PHONY check-env
	$(GO) fmt github.com/ContentMine/ScienceSourceReview

vet: .PHONY
	$(GO) vet github.com/ContentMine/ScienceSourceReview

test: .PHONY vet
	$(GO) test github.com/ContentMine/ScienceSourceReview

get: .PHONY
	$(GIT) submodule update --init

clean: .PHONY
	rm -r bin
	rm -r pkg

.PHONY:

check-env: .PHONY
ifndef GOPATH
	$(error GOPATH is undefined)
endif
