.ONESHELL:
.DEFAULT_GOAL := build
.SHELLFLAGS += -e
MAKEFLAGS += --no-print-directory

# Build values
PKG := ./...
OUT := .
TAGS := json1 fts5
BENCH_COUNT := 3

# Command values
ADDR := :8080

RM := rm
ifeq ($(OS),Windows_NT)
	SHELL = cmd
	RM := del
endif

BUILD_TAGS := $(TAGS)
ifdef TAGS
	BUILD_TAGS := -tags "$(TAGS)"
	BUILD_FLAGS += $(BUILD_TAGS)
endif

ifdef RACE
	BUILD_FLAGS += -race
endif

ifdef DEBUG
	# -N disables all optimisations
	# -l disables inlining
	# See: go tool compile -help
	BUILD_FLAGS += -gcflags "-N -l"
else
	BUILD_FLAGS += -trimpath
endif

ifdef OPTIMISATIONS
	BUILD_FLAGS += -gcflags "$(OPTIMISATIONS)=-m"
endif

ifdef CHECK_BCE
	BUILD_FLAGS += -gcflags "$(CHECK_BCE)=-d=ssa/check_bce"
endif

ifndef DEBUG
	# -s disables the symbol table
	# -w disables DWARF generation
	# See: go tool link -help
	BUILD_FLAGS += -ldflags "-s -w"
endif

ifdef WINDOWSGUI
	BUILD_FLAGS += -ldflags "-H windowsgui"
endif

.PHONY: build
build:
	go build $(BUILD_FLAGS) -o $(OUT) $(PKG)

.PHONY: vet
vet:
	go vet $(BUILD_TAGS) $(PKG)

.PHONY: test
test:
	go test $(BUILD_TAGS) -race -vet off $(PKG)

.PHONY: vulncheck
vulncheck:
	govulncheck $(BUILD_TAGS) $(PKG)

.PHONY: audit
audit: vet test vulncheck

.PHONY: bench
bench:
ifeq ($(PKG),./...)
	@echo Please set the PKG variable to the specific package you want to benchmark
	@echo For example: make bench PKG=./internal/foo
else
	go test $(BUILD_TAGS) -vet off -run no-tests -bench . -count $(BENCH_COUNT) $(PKG)
endif

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: tidy
tidy: fmt
	go mod tidy -v

.PHONY: cover
cover:
	go test $(BUILD_TAGS) -vet off -coverprofile coverage.out $(PKG)
	go tool cover -html=coverage.out
	$(RM) coverage.out

.PHONY: run/web
run/web:
	./web -dev -addr $(ADDR) -behind-secure-proxy -log-style dev
