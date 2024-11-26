# Copyright: Copyright (c) Tao Chen
# Author: Tao Chen <chen.dylane@gmail.com>
# Time: 2024-11-20 12:01:59
import os
import time

def create_write_cache(file_path, size_mb):
    # Convert size from MB to bytes
    size_bytes = size_mb * 1024 * 1024
    # Generate a block of data to write
    data_block = b'0' * 1024 * 1024  # 1MB block of zeros

    with open(file_path, 'wb') as f:
        for _ in range(size_mb):
            f.write(data_block)

    print(f"{size_mb}MB file created at {file_path}")
    #time.sleep(20)

def create_read_cache(file_path, size_mb):
    with open(file_path, 'rb') as f:
        while f.read(1024 * 1024):
            pass
    print(f"{size_mb}MB file created from {file_path}")


def main():
    file_path = 'large_file.dat'
    size_mb = 200
    create_write_cache(file_path, size_mb)

if __name__ == '__main__':
    main()
