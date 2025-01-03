/* SPDX-License-Identifier: (LGPL-2.1 OR BSD-2-Clause) */
#ifndef __OFFCPUTIME_H
#define __OFFCPUTIME_H
#include "vmlinux.h"

#define TASK_COMM_LEN 16
#define SCHED_CACHE_SIZE 512
#define SCHED_CACHE_RECORD_ON 0
#define SCHED_CACHE_RECORD_OFF 1

struct pid_info {
	__u32 pid;
	__u32 tgid;
};

struct user_args {
	__u32 pid;
	__u32 tgid;
	__u32 min_offcpu_ms;
	__u32 max_offcpu_ms;
	__u32 rq_dur_ms;
};

struct waker_t {
	__u32 pid;
	__u32 tgid;
	__u32 t_pid; /* target pid */
	__u32 pad;
	char t_comm[TASK_COMM_LEN];
	int32_t user_stack_id;
	int32_t kern_stack_id;
	char comm[TASK_COMM_LEN];
	__u64 oncpu_ns;
	__u64 offcpu_ns;
	__u64 onrq_ns;
	__u32 offcpu_id;
	__u32 oncpu_id;
	__u64 run_delay_ns;
};

struct target_t {
	__u32 pid;
	__u32 tgid;
	__u32 w_pid; /* waker pid */
	__u32 pad;
	char w_comm[TASK_COMM_LEN];
	int32_t user_stack_id;
	int32_t kern_stack_id;
	char comm[TASK_COMM_LEN];
	__u64 oncpu_ns;
	__u64 offcpu_ns;
	__u64 onrq_ns;
	__u32 offcpu_id;
	__u32 oncpu_id;
	__u64 run_delay_ns;
};

struct perf_event_t {
	struct waker_t waker;
	struct target_t target;
	__u64 dur_ms;
	__u64 rq_dur_ms;
	__u64 ts_ns;
	/* 1: dump, 0: no dump */
	__u32 is_sched_cache_dump;
	__u32 cpu;
};

struct trace_event_t {
	/* waker wake target */
	struct waker_t waker;
	struct target_t target;
};

struct sched_record {
	u32 pid;
	u32 prio;
	char comm[TASK_COMM_LEN];
	u64 ts;
};

struct sched_cached {
	u32 status;
	u32 cpu;
	u32 id;
	u32 pad;
	struct sched_record records[SCHED_CACHE_SIZE];
};

#endif /* __OFFCPUTIME_H */
