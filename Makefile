CLANG ?= clang-14
STRIP ?= llvm-strip-14
OBJCOPY ?= llvm-objcopy-14
CFLAGS := -O2 -g -Wall -Werror $(CFGAGS)
CFLAGS := -ggdb -gdwarf -O2 -Wall -fpie -Wno-unused-variable -Wno-unused-function $(CFLAGS)
PWD := $(shell pwd)
GIT_COMIID := $(shell git rev-parse --short HEAD)
GIT_TAG := $(shell git describe --tags --abbrev=0)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

RELEASE_VERSION := $(GIT_BRANCH)_$(GIT_COMIID)
RELEASE_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
RELEASE_COMMIT := $(GIT_COMIID)
RELEASE_GOVERSION := $(shell go version)
RELEASE_AUTHOR := dylane
GOLDFLAGS :=

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

all: generate libbpf ebpf.o
	@echo "go build $(TARGET)"
	CGO_CFLAGS="$(EXTRA_CGO_CFLAGS)" \
	CGO_LDFLAGS="$(EXTRA_CGO_LDFLAGS)" \
	go build -ldflags "-linkmode external -extldflags '-static' -X 'main.version=$(RELEASE_VERSION)' \
	-X 'main.commitId=$(RELEASE_COMMIT)' -X 'main.releaseTime=$(RELEASE_TIME)' \
	-X 'main.goVersion=$(RELEASE_GOVERSION)' -X 'main.author=$(RELEASE_AUTHOR)'" -o $(TARGET) $(APP_SRC)

clean:
	find . -name "*.o" | xargs rm -f
	#make -C $(LIBBPF) clean
	#make -C $(LIBBCC) clean
