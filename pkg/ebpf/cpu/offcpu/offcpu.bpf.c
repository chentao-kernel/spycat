// SPDX-License-Identifier: GPL-2.0
#include <vmlinux.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_tracing.h>
#include "offcpu.h"

#define PF_KTHREAD		0x00200000	/* I am a kernel thread */
#define MAX_ENTRIES		10240

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__type(key, u32); /* pid */
	__type(value, struct trace_event_t);
	__uint(max_entries, MAX_ENTRIES);
} start SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_STACK_TRACE);
	__uint(key_size, sizeof(u32));
	__uint(value_size, 127 * sizeof(u64));
	__uint(max_entries, MAX_ENTRIES);
} stackmap SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__type(key, u32);
	__type(value, struct user_args);
	__uint(max_entries, 8);
} args_map SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
	__uint(key_size, sizeof(u32));
	__uint(value_size, sizeof(u32));
} perf_map SEC(".maps");

static bool allow_record(u32 tgid, u32 pid,
				u32 min_offtime, u32 max_offtime)
{
	struct user_args *args = NULL;
	u32 index = 0;

	args = bpf_map_lookup_elem(&args_map, &index);
	if (!args)
		return true;

	if (args->tgid == -1 && args->pid == -1)
		return true;

	if (args->tgid != -1 && args->tgid == tgid)
		return true;

	if (args->pid != -1 && args->pid == pid)
		return true;

	if (args->min_offcpu_ms < min_offtime)
		return true;

	if (args->max_offcpu_ms > max_offtime)
		return true;

	return false;
}

SEC("raw_tp/sched_wakeup")
int BPF_PROG(shched_wakeup_hook, struct task_struct *p)
{
	struct trace_event_t *trace_event_p;
	struct trace_event_t trace_event = {0};
	u64 pid_tgid;

	bpf_core_read(&trace_event.target.pid, sizeof(u32), &p->pid);
	bpf_core_read(&trace_event.target.tgid, sizeof(u32), &p->tgid);

	if (!allow_record(trace_event.target.tgid, trace_event.target.pid, 0, -1))
		return 0;

	/* update waker info */
	pid_tgid = bpf_get_current_pid_tgid();
	trace_event_p = bpf_map_lookup_elem(&start, &trace_event.target.pid);
	if (trace_event_p) {
		trace_event_p->waker.pid = (u32)pid_tgid;
		trace_event_p->waker.tgid = pid_tgid >> 32;
		bpf_get_current_comm(&trace_event_p->waker.comm, TASK_COMM_LEN);
		trace_event_p->waker.kern_stack_id = bpf_get_stackid(ctx, &stackmap,
							BPF_F_FAST_STACK_CMP);
		trace_event_p->waker.user_stack_id = bpf_get_stackid(ctx, &stackmap,
							BPF_F_USER_STACK | BPF_F_FAST_STACK_CMP);
			
		/* target on runq time */
		trace_event_p->target.onrq_ns = bpf_ktime_get_ns();
	} else {
		trace_event.waker.pid = (u32)pid_tgid;
		trace_event.waker.tgid = pid_tgid >> 32;
		bpf_get_current_comm(&trace_event.waker.comm, TASK_COMM_LEN);
		bpf_probe_read(&trace_event.waker.t_pid, sizeof(u32), &p->pid);
		bpf_probe_read(trace_event.waker.t_comm, TASK_COMM_LEN, p->comm);
		trace_event.waker.kern_stack_id = bpf_get_stackid(ctx, &stackmap, BPF_F_FAST_STACK_CMP);
		trace_event.waker.user_stack_id = bpf_get_stackid(ctx, &stackmap,
									BPF_F_USER_STACK | BPF_F_FAST_STACK_CMP);
		trace_event.target.onrq_ns = bpf_ktime_get_ns();
		bpf_probe_read(&trace_event.target.pid, sizeof(u32), &p->pid);
		bpf_probe_read(trace_event.target.comm, TASK_COMM_LEN, p->comm);
		/* target first time record on start map */
		bpf_map_update_elem(&start, &trace_event.waker.t_pid, &trace_event, BPF_ANY);
	}

	return 0;
}

SEC("raw_tp/sched_switch")
int BPF_PROG(sched_switch_hook, bool preempt, struct task_struct *prev,
					struct task_struct *next)
{
	struct trace_event_t *trace_event = NULL;
	struct user_args *args = NULL;
	struct perf_event_t perf_event = {0};
	struct pid_info prev_pid = {0};
	struct pid_info next_pid = {0};
	u32 args_map_id = 0;
	bool is_target = false;
	u64 curr_ts;
	u32 cpu_id;
	s32 delta;

	args = bpf_map_lookup_elem(&args_map, &args_map_id);
	if (!args)
		return 0;

	bpf_probe_read(&prev_pid.pid, sizeof(prev_pid.pid), &prev->pid);	
	bpf_probe_read(&prev_pid.tgid, sizeof(prev_pid.tgid), &prev->tgid);	
	bpf_probe_read(&next_pid.pid, sizeof(next_pid.pid), &next->pid);	
	bpf_probe_read(&next_pid.tgid, sizeof(next_pid.tgid), &next->tgid);

	if (args->pid == -1 && args->tgid == -1)
		is_target = true;

	if ((args->pid != -1) && ((args->pid == prev_pid.pid) || (args->pid == next_pid.pid)))
		is_target = true;

	if ((args->tgid != -1) && ((args->tgid == prev_pid.tgid) || (args->tgid == next_pid.tgid)))
		is_target = true;

	if (!is_target)
		return 0;

	trace_event = bpf_map_lookup_elem(&start, &next_pid.pid);
	if (!trace_event)
		return 0;
		
	//bpf_printk("prev_pid%u, prev_tgid:%u\n", prev_pid.pid, prev_pid.tgid);
	//bpf_printk("min:%u, max:%u\n", args->min_offcpu_ms, args->max_offcpu_ms);
	/* target oncpu time */
	curr_ts = bpf_ktime_get_ns();
	cpu_id = bpf_get_smp_processor_id();
	trace_event->target.oncpu_ns = curr_ts;
	trace_event->target.oncpu_id = cpu_id;

	if (trace_event->target.offcpu_ns != 0) {
		delta = (curr_ts - trace_event->target.offcpu_ns) / 1000000;
		if ((delta > 0) && (delta > args->min_offcpu_ms) && (delta < args->max_offcpu_ms)) {
			perf_event.target = trace_event->target;
			perf_event.waker = trace_event->waker;
			perf_event.ts = curr_ts;
			perf_event.offtime_delta = delta;

			/* output event */
			bpf_perf_event_output(ctx, &perf_map, BPF_F_CURRENT_CPU,
							&perf_event, sizeof(perf_event));
		}
	}
	/* update target offcpu info */
	trace_event = bpf_map_lookup_elem(&start, &prev_pid.pid);
	if (trace_event) {
		trace_event->target.offcpu_ns = curr_ts;
		trace_event->target.offcpu_id = cpu_id;
		trace_event->target.kern_stack_id = bpf_get_stackid(ctx, &stackmap,
								BPF_F_FAST_STACK_CMP);
		trace_event->target.user_stack_id = bpf_get_stackid(ctx, &stackmap,
								BPF_F_USER_STACK | BPF_F_FAST_STACK_CMP);
	} else {
		struct trace_event_t event = {0};
		event.target.pid = prev_pid.pid;
		event.target.tgid = prev_pid.tgid;
		event.target.offcpu_ns = curr_ts;
		event.target.offcpu_id = cpu_id;
		bpf_probe_read(event.target.comm, TASK_COMM_LEN, prev->comm);
		event.target.kern_stack_id = bpf_get_stackid(ctx, &stackmap,
								BPF_F_FAST_STACK_CMP);
		event.target.user_stack_id = bpf_get_stackid(ctx, &stackmap,
								BPF_F_USER_STACK | BPF_F_FAST_STACK_CMP);
		bpf_map_update_elem(&start, &event.target.pid, &event, BPF_ANY);
	}
	
	return 0;
}
char LICENSE[] SEC("license") = "GPL";
