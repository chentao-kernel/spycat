#ifndef __ONCPU_H
#define __ONCPU_H

#define PERF_MAX_STACK_DEPTH 127
#define PROFILE_MAPS_SIZE 16384

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

struct profile_key_t {
	__u32 pid;
	__s64 kern_stack;
	__s64 user_stack;
	char comm[16];
};

struct profile_value_t {
	__u64 counts;
	__u64 deltas;
};

struct profile_bss_args_t {
	__u32 tgid_filter; // 0 => profile everything
	int cpu_filter;
	int space_filter; // 0 user
};

struct net_args {
	int dport;
	int sport;
	int delay;
};

struct trace_info {
	__u64 ts;
	__u32 pid;
	__u32 cpu1;
	__u32 cpu2;
	__u16 sport;
	__u16 dport;
	__u32 ref;
	__s64 kern_stack;
	__s64 user_stack;
	char comm[16];
};

#endif /* __ONCPU_H */
