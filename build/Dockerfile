FROM golang:1.20-alpine3.16

RUN sed -i "s/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g" /etc/apk/repositories

RUN apk update && apk upgrade && \
    apk add cmake --no-cache binutils clang make git zstd gcc g++ libc-dev musl-dev bash zlib-dev elfutils-dev libelf-static zlib-static openssh llvm \
    linux-headers

RUN go env -w GOPROXY="https://proxy.golang.com.cn,direct"
