.PHONY: build release clean test

DIST := dist
RELEASE := $(DIST)/release
BINARY := paul-envs
CMD := ./cmd/paul-envs

TARGETS := linux-amd64 linux-arm64 windows-amd64 windows-arm64 macos-amd64 macos-arm64

VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -s -w -X main.version=$(VERSION)

export CGO_ENABLED=0

build:
	go build -o $(DIST)/$(BINARY) $(CMD)

release:
	@rm -rf $(RELEASE) && mkdir -p $(RELEASE)
	@for t in $(TARGETS); do \
		os=$${t%-*}; \
		arch=$${t#*-}; \
		[ "$$os" = "macos" ] && os="darwin"; \
		echo "building $$t …"; \
		out="$(BINARY)-$$t"; \
		[ "$$os" = "windows" ] && out="$$out.exe"; \
		GOOS=$$os GOARCH=$$arch \
			go build -ldflags "$(LDFLAGS)" -trimpath \
			-o $(RELEASE)/$$out $(CMD); \
		(cd $(RELEASE) && zip -q "$${out%.exe}.zip" "$$out"); \
		rm $(RELEASE)/$$out; \
	done
	@echo "done → $(RELEASE)/*.zip"

clean:
	rm -rf $(DIST)

test:
	CGO_ENABLED=1 go test ./... -race -count=1
