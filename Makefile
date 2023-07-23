.DEFAULT_GOAL := build
MAKEFLAGS += --no-print-directory

# Build values
PKG := ./...
OUT := .
TAGS := json1 fts5

# Command values
ADDR := :8080

rm := rm
ifeq ($(OS),Windows_NT)
	SHELL = cmd
	rm := del
endif

build_tags := $(TAGS)
ifdef TAGS
	build_tags := -tags "$(TAGS)"
	build_flags += $(build_tags)
endif

ifdef RACE
	build_flags += -race
endif

ifdef DEBUG
	# -N disables all optimisations
	# -l disables inlining
	# See: go tool compile -help
	build_flags += -gcflags "-N -l"
else
	build_flags += -trimpath
endif

ifdef OPTIMISATIONS
	build_flags += -gcflags "$(OPTIMISATIONS)=-m"
endif

ifdef CHECK_BCE
	build_flags += -gcflags "$(CHECK_BCE)=-d=ssa/check_bce"
endif

ifndef DEBUG
	# -s disables the symbol table
	# -w disables DWARF generation
	# See: go tool link -help
	build_flags += -ldflags "-s -w"
endif

ifdef WINDOWSGUI
	build_flags += -ldflags "-H windowsgui"
endif

.PHONY: build
build:
	go build $(build_flags) -o $(OUT) $(PKG)

.PHONY: vet
vet:
	go vet $(build_tags) $(PKG)

.PHONY: test
test:
	go test $(build_tags) -race -vet off $(PKG)

.PHONY: vulncheck
vulncheck:
	govulncheck $(build_tags) $(PKG)

.PHONY: audit
audit: vet test vulncheck

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: tidy
tidy: fmt
	go mod tidy -v

.PHONY: cover
cover:
	go test $(build_tags) -vet off -coverprofile coverage.out $(PKG)
	go tool cover -html=coverage.out
	$(rm) coverage.out

.PHONY: run/web
run/web:
	./web -dev -addr $(ADDR) -behind-secure-proxy -log-style dev
