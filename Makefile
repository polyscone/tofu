.DEFAULT_GOAL := build
.SHELLFLAGS += -e
MAKEFLAGS += --no-print-directory

GOROOT := $(subst \,/,$(shell go env GOROOT))
PKG := ./...
OUT := .
TAGS := json1 fts5
BENCH_COUNT := 3
DATA := ./.data

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

	ifeq ($(OS),Windows_NT)
		ifneq ($(PKG),./...)
			# On Windows disassembly in tools like pprof aren't supported
			# in position-independent executables (PIE), which is the default
			# build mode for Go
			#
			# Because of thise Windows has to set its build mode to exe, but
			# the exe build mode can only be used in builds where there is one
			# main function, so we only include the flag when we're not building
			# all packages
			BUILD_FLAGS += -buildmode exe
		endif
	endif
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

.PHONY: test
test:
	go test $(BUILD_TAGS) -race -vet off $(PKG)

.PHONY: audit
audit:
	go mod tidy -v
	go mod verify
	go fmt ./...
	go vet $(BUILD_TAGS) ./...
	go test $(BUILD_TAGS) -race -vet off ./...

.PHONY: vulncheck
vulncheck:
	govulncheck $(BUILD_TAGS) ./...

.PHONY: bench
bench:
ifeq ($(PKG),./...)
	@echo Please set the PKG variable to the specific package you want to benchmark
	@echo For example: make bench PKG=./internal/foo
else
	go test $(BUILD_TAGS) -vet off -run no-tests -bench . -count $(BENCH_COUNT) $(PKG)
endif

.PHONY: cover
cover:
	go test $(BUILD_TAGS) -vet off -coverprofile coverage.out $(PKG)
	go tool cover -html=coverage.out

GEN_CERT_HOST := localhost
.PHONY: gen/cert
gen/cert:
	cd $(DATA) && \
	go run $(GOROOT)/src/crypto/tls/generate_cert.go -rsa-bits 2048 -host "$(GEN_CERT_HOST)"

WEB_DEV_ADDR := :8080
WEB_DEV_DEBUG_ADDR := :8081
override WEB_DEV_FLAGS := -dev -addr $(WEB_DEV_ADDR) -debug-addr $(WEB_DEV_DEBUG_ADDR) -log-style dev $(WEB_DEV_FLAGS)
.PHONY: web/dev
web/dev:
	$(CURDIR)/web $(WEB_DEV_FLAGS)
