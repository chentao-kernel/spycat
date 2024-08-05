LIBBPF_VERSION ?= v1.2.0
LIBBPF_SRC = $(CURDIR)/src/src
LIBBPF_DESTDIR = $(CURDIR)/lib/include

.PHONY: $(LIBBPF_SRC)
$(LIBBPF_SRC):
	mkdir -p $(LIBBPF_DESTDIR)
	test -d src || git clone https://github.com/libbpf/libbpf src
	cd src && git checkout $(LIBBPF_VERSION)

# libbpf uapi
.PHONY: libbpf-uapi
libbpf-uapi: $(LIBBPF_SRC)
# UAPI headers can be installed by a different package so they're not installed
# in by (libbpf) install rule.
	UAPIDIR=$(LIBBPF_DESTDIR) \
                $(MAKE) -C $(LIBBPF_SRC) install_uapi_headers

.PHONY: libbpf
libbpf: libbpf-uapi
	PREFIX=$(shell pwd)/lib make -C src/src -j16 install

.PHONY: clean
clean:
	rm -rf src lib
