/*
 * Copyright: Copyright (c) Tao Chen
 * Author: Tao Chen <chen.dylane@gmail.com>
 */

#ifndef __CACHESTAT_H
#define __CACHESTAT_H

#define TASK_COMM_LEN 16
#define CACHE_FILE_SIZE 64
#define CACHESTAT_MAP_SIZE 10240

typedef signed char __s8;
typedef unsigned char __u8;
typedef short int __s16;
typedef short unsigned int __u16;
typedef int __s32;
typedef unsigned int __u32;
typedef long long int __s64;
typedef long long unsigned int __u64;
typedef __s8 s8;
typedef __u8 u8;
typedef __s16 s16;
typedef __u16 u16;
typedef __s32 s32;
typedef __u32 u32;
typedef __s64 s64;
typedef __u64 u64;

/* TODO: */
struct user_args {
	__u32 pid;
	__u32 cache_type;
};

enum CACHE_TYPE {
	READ_WRITE_CACHE = 0,
	READ_CACHE = 1,
	WRITE_CACHE = 2,
};
struct cache_key {
	__u64 pid;
	const unsigned char *file;
};

struct cache_info {
	__u32 pid;
	__u32 pad;
	char comm[TASK_COMM_LEN];
	char file[CACHE_FILE_SIZE];
	__u64 read_size;
	__u64 write_size;
};

#endif // !__CACHESTAT_H
