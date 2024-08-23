.DEFAULT_GOAL := build
.SHELLFLAGS += -e
MAKEFLAGS += --no-print-directory

GOROOT := $(subst \,/,$(shell go env GOROOT))
PKG := ./...
OUT := .
TAGS :=
GCFLAGS :=
LDFLAGS :=
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
	GCFLAGS += -N -l

	ifeq ($(OS),Windows_NT)
		ifneq ($(PKG),./...)
			# On Windows disassembly in tools like pprof aren't supported
			# in position-independent executables (PIE), which is the default
			# build mode for Go
			#
			# Because of this Windows has to set its build mode to exe, but
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
	GCFLAGS += $(OPTIMISATIONS)=-m
endif

ifdef CHECK_BCE
	GCFLAGS += $(CHECK_BCE)=-d=ssa/check_bce
endif

ifdef STRIP_SYMBOLS
	# -s disables the symbol table
	# -w disables DWARF generation
	# See: go tool link -help
	LDFLAGS += -s -w
endif

ifdef WINDOWS_GUI
	LDFLAGS += -H windowsgui
endif

ifdef GCFLAGS
	BUILD_FLAGS += -gcflags "$(GCFLAGS)"
endif

ifdef LDFLAGS
	BUILD_FLAGS += -ldflags "$(LDFLAGS)"
endif

.PHONY: build
build:
	go build $(BUILD_FLAGS) -o $(OUT) $(PKG)

TEST_BUILD_TAGS := $(BUILD_TAGS)
TEST_BUILD_TAGS += -race -vet off

.PHONY: test
test:
	go test $(TEST_BUILD_TAGS) $(PKG)

.PHONY: audit
audit:
	go mod tidy -v
	go mod verify
	go fmt ./...
	go vet $(BUILD_TAGS) ./...
	go test $(TEST_BUILD_TAGS) ./...

.PHONY: bench
bench:
ifeq ($(PKG),./...)
	@echo Please set the PKG variable to the specific package you want to benchmark
	@echo For example: make bench PKG=./foo
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

HTTPD_DEV_ADDR := :8080
HTTPD_DEV_DEBUG_ADDR := :8081
override HTTPD_DEV_FLAGS := -dev -addr $(HTTPD_DEV_ADDR) -debug-addr $(HTTPD_DEV_DEBUG_ADDR) -log-style dev $(HTTPD_DEV_FLAGS)
.PHONY: httpd/dev
httpd/dev:
	$(CURDIR)/httpd $(HTTPD_DEV_FLAGS)
