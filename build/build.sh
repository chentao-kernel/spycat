#!/bin/bash

#refrence:https://www.kancloud.cn/woshigrey/docker/935037
# run arm docker on x86 host
# docker run --rm --privileged multiarch/qemu-user-static --reset -p yes

read -r -d '' USAGE << EOF || true
usage: ./build.sh [-h|--help -b|--build -c|--compile -t|--tar -V|--bin_ver -p|--proxy]
	-b              build image
	-c              compile
	-t              tar binary
        -V 0.1.2 -t     tar binary with 0.1.2 version
	-p 		add docker proxy
	-u 		add docker user
EOF

die() {
    echo "ERROR: ${*}"
    exit 1
}

info() {
    echo "INFO: ${*}"
}

VERSION="0.1.0"
CUR_PATH=$(pwd)
SOURCE_PATH=$(dirname "$CUR_PATH")
IMAGE="spycat"
BIN_VER="0.1.0"

build_image() {
        echo "build image start"
        if [[ -f "${CUR_PATH}/Dockerfile" ]]; then
                docker build -t "${IMAGE}" -f Dockerfile .
        else
                die "docker file no found"
        fi

        if [[ -n "$(docker images | grep ${IMAGE})" ]]; then
                info "build image success"
        else
                die "build image failed"
        fi
}

compile_bin() {
        info "compile bin start"
        image_id=$(docker images | grep ${IMAGE} | awk '{print $3}')
        if [[ -n "${image_id}" ]];then
                container_id=$(docker ps | grep ${IMAGE} | awk '{print $1}')
                if [[ -n "${container_id}" ]];then
                        docker exec -it ${container_id} bash -c "cd /root/spycat;make all"
                else
                        docker run -itd --name ${IMAGE} --privileged -v ${SOURCE_PATH}:/root/spycat --net host ${image_id}
                        container_id=$(docker ps | grep ${IMAGE} | awk '{print $1}')
                        docker exec -it ${container_id} bash -c "cd /root/spycat;make all"
                fi
        else
                die "image no found:${image_id}"
        fi

        if [[ -f "${SOURCE_PATH}/spycat" ]]; then
                info "compile bin success"
        else
                die "compile bin failed"
        fi
}

docker_user() {
	sudo groupadd docker
	sudo usermod -aG docker $USER
}

docker_proxy() {
	sudo mkdir -p /etc/docker
	sudo tee /etc/docker/daemon.json <<-'EOF'
	{
		"registry-mirrors": [
		"https://docker.m.daocloud.io",
		"https://dockerproxy.com",
		"https://docker.nju.edu.cn",
		"https://docker.mirrors.ustc.edu.cn"
		]
	}
	EOF
	sudo systemctl daemon-reload
	sudo systemctl restart docker
}

tar_bin() {
        info "tar bin start"
        if [[ -f "${SOURCE_PATH}/spycat" ]]; then
                cp ${SOURCE_PATH}/spycat .
                tar -zcvf spycat_${BIN_VER}.tar.gz spycat
                rm spycat
		info "tar bin success"
        else 
                die "tar bin not found"
        fi
}

parse_arguments() {
        while [[ $# -gt 0 ]];do
                case $1 in
                        -v|--version)
                                echo $VERSION
                                shift 1
                                ;;
                        -V|--bin_ver)
                                BIN_VER=$2
                                echo "BIN_VER:${BIN_VER}"
                                shift 2
                                ;;
                        -h|--help)
                                echo "$USAGE"
                                exit 0
                                ;;
                        -t|--tar)
                                tar_bin
                                shift 1
                                ;;
                        -b|--build)
                                build_image
                                shift 1
                                ;;
                        -c|--compile)
                                compile_bin
                                shift 1
                                ;;
                        -p|--proxy)
				docker_proxy
				shift 1
				;;
			-u|--user)
				docker_user
				shift 1
				;;
                        *)
                                echo "$USAGE"
                                exit 1
                                ;;
                esac
        done
}

parse_arguments "$@"
