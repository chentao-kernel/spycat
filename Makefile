
.ONESHELL:
SHELL = /bin/bash

CLANG ?= clang
STRIP ?= llvm-strip
OBJCOPY ?= llvm-objcopy
CFLAGS := -O2 -g -Wall -Werror $(CFGAGS)
CFLAGS := -ggdb -gdwarf -O2 -Wall -fpie -Wno-unused-variable -Wno-unused-function $(CFLAGS)
PWD := $(shell pwd)
GIT_COMIID := $(shell git rev-parse --short HEAD)
GIT_TAG := $(shell git describe --tags --abbrev=0)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
CLANG_FMT := clang-format-14

RELEASE_VERSION := $(GIT_BRANCH)_$(GIT_COMIID)
RELEASE_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
RELEASE_COMMIT := $(GIT_COMIID)
RELEASE_GOVERSION := $(shell go version)
RELEASE_AUTHOR := dylane
GOLDFLAGS :=
REVIVE := revive

EXTRA_CGO_CFLAGS := -I$(abspath lib/libbpf/lib/include) \
        -I$(abspath lib/bcc/lib/include/bcc_syms)
EXTRA_CGO_LDFLAGS := -L$(abspath lib/libbpf/lib/lib64) -lbpf \
                -L$(abspath lib/bcc/lib/lib) -lbcc-syms -lstdc++ -lelf -lz
TARGET ?= spycat
EBPF_SRC := $(PWD)/pkg/ebpf
APP_SRC := $(PWD)/cmd/spycat/main.go 
LIBBPF := $(PWD)/lib/libbpf
LIBBCC := $(PWD)/lib/bcc
INCLUDE := -I$(PWD)/pkg/ebpf/headers

CLANG_COMPILE := $(CLANG) $(CFLAGS) $(INCLUDE) -target bpf -D__TARGET_ARCH_x86 $(BPFAPI)


.PHONY: all generate

generate: export BPF_CLANG := $(CLANG)
generate: export BPF_CFLAGS := $(CFLAGS)
generate:
	go generate $(EBPF_SRC)/uprobe/uprobe.go


libbpf:
	make -C $(LIBBPF)
	make -C $(LIBBCC)

ebpf.o: libbpf
	$(CLANG_COMPILE) -c $(EBPF_SRC)/cpu/offcpu/offcpu.bpf.c -o $(EBPF_SRC)/cpu/offcpu/offcpu.bpf.o
	$(CLANG_COMPILE) -c $(EBPF_SRC)/cpu/oncpu/oncpu.bpf.c -o $(EBPF_SRC)/cpu/oncpu/oncpu.bpf.o

all: generate libbpf ebpf.o
	@echo "go build $(TARGET)"
	CGO_CFLAGS="$(EXTRA_CGO_CFLAGS)" \
	CGO_LDFLAGS="$(EXTRA_CGO_LDFLAGS)" \
	go build -ldflags "-linkmode external -extldflags '-static' -X 'main.version=$(RELEASE_VERSION)' \
	-X 'main.commitId=$(RELEASE_COMMIT)' -X 'main.releaseTime=$(RELEASE_TIME)' \
	-X 'main.goVersion=$(RELEASE_GOVERSION)' -X 'main.author=$(RELEASE_AUTHOR)'" -o $(TARGET) $(APP_SRC)

# fmt-check clone from libbpfgo

C_FILES_TO_BE_CHECKED = $(shell find ./pkg -regextype posix-extended -regex '.*\.(h|c)' ! -regex '.*(headers|output)\/.*' | xargs)

fmt-check:
	@errors=0
	echo "Checking C and eBPF files and headers formatting..."
	$(CLANG_FMT) --dry-run -i $(C_FILES_TO_BE_CHECKED) > /tmp/check-c-fmt 2>&1
	clangfmtamount=$$(cat /tmp/check-c-fmt | wc -l)
	if [[ $$clangfmtamount -ne 0 ]]; then
		head -n30 /tmp/check-c-fmt
		errors=1
	fi
	rm -f /tmp/check-c-fmt
#
	if [[ $$errors -ne 0 ]]; then
		echo
		echo "Please fix formatting errors above!"
		echo "Use: $(MAKE) fmt-fix target".
		echo
		exit 1
	fi

# fmt-fix

fmt-fix:
	@echo "Fixing C and eBPF files and headers formatting..."
	$(CLANG_FMT) -i --verbose $(C_FILES_TO_BE_CHECKED)

# lint-check
#
.PHONY: lint-check
lint-check:
	@errors=0
	echo "Linting golang code..."
	$(REVIVE) -config .revive.toml ./...

clean:
	find . -name "*.o" | xargs rm -f
	#make -C $(LIBBPF) clean
	#make -C $(LIBBCC) clean
