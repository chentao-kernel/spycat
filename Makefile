CLANG ?= clang-14
STRIP ?= llvm-strip-14
OBJCOPY ?= llvm-objcopy-14
CFLAGS := -O2 -g -Wall -Werror $(CFGAGS)
CFLAGS := -ggdb -gdwarf -O2 -Wall -fpie -Wno-unused-variable -Wno-unused-function $(CFLAGS)
PWD := $(shell pwd)

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
	go build -o $(TARGET) $(APP_SRC)

clean:
	find . -name "*.o" | xargs rm -f
	#make -C $(LIBBPF) clean
	#make -C $(LIBBCC) clean
