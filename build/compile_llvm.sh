#!/bin/bash

git clone https://github.com/llvm/llvm-project.git
cd llvm-project
git checkout llvm-10.0.0  # 切换到 10.0.0 版本

mkdir build
cd build
cmake -G "Unix Makefiles" -DLLVM_ENABLE_PROJECTS="clang" -DCMAKE_BUILD_TYPE=Release ../llvm
make -j$(nproc)

sudo make install

#check
clang --version
