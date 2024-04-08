/* SPDX-License-Identifier: (LGPL-2.1 OR BSD-2-Clause) */
/*
 * Create: Fri Mar 29 23:05:46 2024
 */
#ifndef __FUTEXCTN_H
#define __FUTEXCTN_H

#define TASK_COMM_LEN 16
#define MAX_SLOTS 24

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

struct hist_key {
	__u64 pid_tgid;
	__u64 uaddr;
	int user_stack_id;
};

struct user_args {
	__u32 targ_pid;
	__u32 targ_tid;
	/* lock addr */
	__u64 targ_lock;
	bool stack;
	__u32 min_dur_ms;
	__u32 max_dur_ms;
	/* user count hold the same lock */
	__u32 max_lock_hold_users;
};
// for single task
struct hist {
	__u32 slots[MAX_SLOTS];
	char comm[TASK_COMM_LEN];
	__u64 pid_tgid;
	__u64 uaddr;
	int user_stack_id;
	__u32 pad;
	__u64 contended;
	__u64 total_elapsed;
	__u64 min_dur;
	__u64 max_dur;
	__u64 delta_dur;
	__u64 max_dur_ts;
};

// for single lock
struct lock_stat {
	uint32_t user_cnt;
	uint32_t max_user_cnt;
	__u64 uaddr;
	char comm[TASK_COMM_LEN];
	__u64 pid_tgid;
	__u64 ts;
};

#endif /* FUTEXCTN_H_ */
