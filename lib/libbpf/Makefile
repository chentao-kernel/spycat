LIBBPF_VERSION ?= v0.8.1

.PHONY: libbpf
libbpf:
	test -d src || git clone http://github.com/libbpf/libbpf src
	cd src && git checkout $(LIBBPF_VERSION)
	PREFIX=$(shell pwd)/lib make -C src/src -j16 install
	cd src/src && make install

.PHONY: clean
clean:
	rm -rf src lib
