/*
 * Create: Tue May 21 14:42:30 2024
 */

#include <vmlinux.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>
#include "syscall.h"

#define MAX_ENTRIES 10240

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, MAX_ENTRIES);
	__type(key, u32);
	__type(value, struct syscall_event);
} start SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, 8);
	__type(key, u32);
	__type(value, struct user_args);
} args_map SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_STACK_TRACE);
	__uint(key_size, sizeof(u32));
	__uint(value_size, 127 * sizeof(u64));
	__uint(max_entries, MAX_ENTRIES);
} stack_map SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
	__uint(key_size, sizeof(u32));
	__uint(value_size, sizeof(u32));
} perf_map SEC(".maps");

SEC("tracepoint/raw_syscalls/sys_enter")
int sys_enter(struct trace_event_raw_sys_enter *args)
{
	struct syscall_event event = { 0 };
	struct user_args *user_args = NULL;
	u64 pid_tgid = bpf_get_current_pid_tgid();
	pid_t pid = pid_tgid >> 32;
	u32 tid = pid_tgid;
	u32 zero = 0, syscall_id;

	user_args = bpf_map_lookup_elem(&args_map, &zero);
	if (!user_args)
		return 0;

	syscall_id = BPF_CORE_READ(args, id);
	if (user_args->pid && user_args->pid != pid)
		return 0;
	if (user_args->tid && user_args->tid != tid)
		return 0;
	if (user_args->syscall_id != -1 && user_args->syscall_id != syscall_id)
		return 0;

	event.ts_ns = bpf_ktime_get_ns();
	event.pid = pid;
	event.tid = tid;
	bpf_map_update_elem(&start, &tid, &event, 0);

	return 0;
}

SEC("tracepoint/raw_syscalls/sys_exit")
int sys_exit(struct trace_event_raw_sys_exit *args)
{
	u64 pid_tgid = bpf_get_current_pid_tgid();
	struct user_args *user_args = NULL;
	struct syscall_event *eventp = NULL;
	struct syscall_event event = { 0 };
	pid_t pid = pid_tgid >> 32;
	u64 dur = 0;
	u32 tid = pid_tgid;
	u32 zero = 0, syscall_id;

	/* this happens when there is an interrupt */
	if (args->id == -1)
		return 0;

	user_args = bpf_map_lookup_elem(&args_map, &zero);
	if (!user_args)
		return 0;

	syscall_id = BPF_CORE_READ(args, id);
	if (user_args->pid && user_args->pid != pid)
		return 0;
	if (user_args->tid && user_args->tid != tid)
		return 0;
	if (user_args->syscall_id != -1 && user_args->syscall_id != syscall_id)
		return 0;

	eventp = bpf_map_lookup_elem(&start, &tid);
	if (!eventp)
		return 0;

	dur = bpf_ktime_get_ns() - eventp->ts_ns;
	if (dur / 1000000 >= user_args->min_syscall_ms &&
	    dur / 1000000 <= user_args->max_syscall_ms) {
		event = *eventp;
		event.dur_us = dur / 1000;
		if (user_args->stack) {
			event.u_stack_id = bpf_get_stackid(
				args, &stack_map,
				BPF_F_USER_STACK | BPF_F_FAST_STACK_CMP);
			event.k_stack_id = bpf_get_stackid(
				args, &stack_map, BPF_F_FAST_STACK_CMP);
		}
		event.syscall_id = syscall_id;
		bpf_get_current_comm(event.comm, sizeof(event.comm));
		bpf_perf_event_output(args, &perf_map, BPF_F_CURRENT_CPU,
				      &event, sizeof(event));
	}

	return 0;
}

char LICENSE[] SEC("license") = "GPL";
