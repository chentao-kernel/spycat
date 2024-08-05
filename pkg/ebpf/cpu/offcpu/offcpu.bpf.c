// SPDX-License-Identifier: GPL-2.0
#include "offcpu.h"
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <vmlinux.h>

#define PF_KTHREAD 0x00200000 /* I am a kernel thread */
#define MAX_ENTRIES 10240

#define NO_REOCED 0
#define RECOED (1 << 1)
#define RECOED_SCHE_CACHE (1 << 2)

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__type(key, u32); /* pid */
	__type(value, struct trace_event_t);
	__uint(max_entries, MAX_ENTRIES);
} start SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__type(key, u32);
	__type(value, struct sched_cached);
	__uint(max_entries, 1);
} sched_cache_map SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__type(key, u32);
	__type(value, struct perf_event);
	__uint(max_entries, 1);
} perf_event_cache_map SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__type(key, u32);
	__type(value, struct sched_cached);
	__uint(max_entries, 1024);
} sched_cache_backup_map SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_STACK_TRACE);
	__uint(key_size, sizeof(u32));
	__uint(value_size, 127 * sizeof(u64));
	__uint(max_entries, MAX_ENTRIES);
} stack_map SEC(".maps");

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

static bool is_target_task(u32 tgid, u32 pid)
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

	return false;
}

SEC("raw_tp/sched_wakeup")
int BPF_PROG(shched_wakeup_hook, struct task_struct *p)
{
	struct trace_event_t *ep = NULL;
	struct trace_event_t event = { 0 };
	u64 pid_tgid, ts;

	bpf_core_read(&event.target.pid, sizeof(u32), &p->pid);
	bpf_core_read(&event.target.tgid, sizeof(u32), &p->tgid);

	if (!is_target_task(event.target.tgid, event.target.pid))
		return 0;
	bpf_probe_read(event.target.comm, TASK_COMM_LEN, p->comm);
	/* update waker info */
	pid_tgid = bpf_get_current_pid_tgid();
	ts = bpf_ktime_get_ns();

	ep = bpf_map_lookup_elem(&start, &event.target.pid);
	if (ep) {
		ep->waker.pid = (u32)pid_tgid;
		ep->waker.tgid = pid_tgid >> 32;
		bpf_get_current_comm(ep->waker.comm, TASK_COMM_LEN);
		ep->waker.kern_stack_id =
				bpf_get_stackid(ctx, &stack_map, BPF_F_FAST_STACK_CMP);
		ep->waker.user_stack_id = bpf_get_stackid(
				ctx, &stack_map,
				BPF_F_USER_STACK | BPF_F_FAST_STACK_CMP);

		/* target on runq time */
		ep->target.onrq_ns = ts;
		ep->waker.offcpu_ns = ts;
	} else {
		event.waker.pid = (u32)pid_tgid;
		event.waker.tgid = pid_tgid >> 32;
		bpf_get_current_comm(&event.waker.comm, TASK_COMM_LEN);

		event.waker.kern_stack_id =
			bpf_get_stackid(ctx, &stack_map, BPF_F_FAST_STACK_CMP);
		event.waker.user_stack_id = bpf_get_stackid(
			ctx, &stack_map,
			BPF_F_USER_STACK | BPF_F_FAST_STACK_CMP);
		event.target.onrq_ns = ts;
		event.waker.offcpu_ns = ts;
		event.waker.t_pid = event.target.pid;
		bpf_probe_read(event.waker.t_comm, TASK_COMM_LEN, p->comm);
		/* target first time record on start map */
		bpf_map_update_elem(&start, &event.target.pid, &event, BPF_ANY);
	}

	return 0;
}

static __always_inline void sched_cache_dump(void *ctx, u32 cpu)
{
	struct sched_cached *cache = NULL;
	u32 zero = 0, id = cpu;

	cache = bpf_map_lookup_elem(&sched_cache_map, &zero);
	if (cache) {
		cache->status = SCHED_CACHE_RECORD_OFF;
		// libbpfgo will lost events
		//bpf_perf_event_output(ctx, &perf_map, BPF_F_CURRENT_CPU, cache, sizeof(struct sched_cached));
		bpf_map_update_elem(&sched_cache_backup_map, &id, cache,
				    BPF_ANY);
		cache->status = SCHED_CACHE_RECORD_ON;
	}
}

static __always_inline void sched_cache_update(struct task_struct *p, u64 ts,
					       u32 cpu)
{
	struct sched_cached *cache = NULL;
	u32 zero = 0;
	u32 id;

	cache = bpf_map_lookup_elem(&sched_cache_map, &zero);
	if (!cache)
		return;

	id = cache->id % SCHED_CACHE_SIZE;
	if (id < SCHED_CACHE_SIZE && cache->status == SCHED_CACHE_RECORD_ON) {
		cache->cpu = cpu;
		bpf_probe_read(&cache->records[id].pid, sizeof(u32), &p->pid);
		cache->records[id].ts = ts;
		bpf_probe_read(&cache->records[id].prio, sizeof(u32), &p->prio);
		bpf_probe_read(&cache->records[id].comm, TASK_COMM_LEN,
			       p->comm);
		cache->id++;
	}
}

//__attribute__((noinline)) static void update_new_task_info(u64 ts, u32 cpu, struct task_struct *task)
static void update_new_task_info(u64 ts, u32 cpu, struct task_struct *task)
{
	struct trace_event_t event = { 0 };
	/*
	* some new task fork and give up cpu, here we reocrd it.
	*/
	event.target.offcpu_ns = ts;
	event.target.offcpu_id = cpu;
	event.target.pid = BPF_CORE_READ(task, pid);
	event.target.tgid = BPF_CORE_READ(task, tgid);
	bpf_core_read(event.target.comm, TASK_COMM_LEN, &task->comm);

	event.target.run_delay_ns = BPF_CORE_READ(task, sched_info.run_delay);
	bpf_map_update_elem(&start, &event.target.pid, &event, BPF_ANY);
}

// todo isra.0 compile issue
SEC("kprobe/finish_task_switch")
int sched_switch_hook(struct pt_regs *ctx)
{
	struct trace_event_t *ep = NULL;
	struct user_args *args = NULL;
	struct perf_event_t *perf_event = NULL;
	struct pid_info prev_pid = { 0 }, cur_pid = { 0 };
	struct task_struct *pre_task = NULL, *cur_task = NULL;
	u32 args_map_id = 0, cpu_id;
	u64 curr_ts, cur_task_run_delay_ns;
	s32 delta, runq_dur;

	args = bpf_map_lookup_elem(&args_map, &args_map_id);
	if (!args)
		return 0;

	cur_task = (struct task_struct *)bpf_get_current_task();
	cur_pid.pid = BPF_CORE_READ(cur_task, pid);
	cur_pid.tgid = BPF_CORE_READ(cur_task, tgid);

	pre_task = (struct task_struct *)PT_REGS_PARM1_CORE(ctx);
	prev_pid.tgid = BPF_CORE_READ(pre_task, tgid);
	prev_pid.pid = BPF_CORE_READ(pre_task, pid);
	curr_ts = bpf_ktime_get_ns();
	cpu_id = bpf_get_smp_processor_id();

	if (args->rq_dur_ms) {
		sched_cache_update(pre_task, curr_ts, cpu_id);
	}

	/* record prev task info */
	if (is_target_task(prev_pid.tgid, prev_pid.pid)) {
		ep = bpf_map_lookup_elem(&start, &prev_pid.pid);
		if (!ep) {
			update_new_task_info(curr_ts, cpu_id, pre_task);
			return 0;
		}
		ep->target.offcpu_ns = curr_ts;
		ep->target.offcpu_id = cpu_id;
		ep->target.run_delay_ns = BPF_CORE_READ(pre_task, sched_info.run_delay);
		return 0;
	}

	/* record current task info */
	if (!is_target_task(cur_pid.tgid, cur_pid.pid))
		return 0;

	ep = bpf_map_lookup_elem(&start, &cur_pid.pid);
	if (!ep)
		return 0;
	/* target oncpu time */
	ep->target.oncpu_id = cpu_id;
	bpf_core_read(&cur_task_run_delay_ns, sizeof(u64),
		      &cur_task->sched_info.run_delay);
	ep->target.oncpu_ns = curr_ts;
	delta = (curr_ts - ep->target.offcpu_ns) / 1000000;
	if ((delta >= args->min_offcpu_ms) &&
		(delta <= args->max_offcpu_ms) &&
		/*
		 * if offcpu_ns = 0, means we do not trace wake and sched_switch
		 * two step completely, so we ignore it.
		 */
		ep->target.offcpu_ns != 0) {
		perf_event = bpf_map_lookup_elem(&perf_event_cache_map, &args_map_id);
		if (!perf_event)
			return 0;

		runq_dur = (cur_task_run_delay_ns -
					ep->target.run_delay_ns) / 1000000;
		perf_event->target = ep->target;
		perf_event->target.kern_stack_id = bpf_get_stackid(
			ctx, &stack_map, BPF_F_FAST_STACK_CMP);
		perf_event->target.user_stack_id = bpf_get_stackid(
			ctx, &stack_map,
			BPF_F_USER_STACK | BPF_F_FAST_STACK_CMP);
		bpf_get_current_comm(perf_event->target.comm,
						TASK_COMM_LEN);
		perf_event->waker = ep->waker;
		perf_event->ts_ns = curr_ts;
		perf_event->dur_ms = delta;
		perf_event->rq_dur_ms = runq_dur;

		/* output task occupy cpu */
		if (runq_dur > args->rq_dur_ms &&
			args->rq_dur_ms != 0) {
			sched_cache_dump(ctx, cpu_id);
			perf_event->is_sched_cache_dump = 1;
			perf_event->cpu = cpu_id;
		}
		/* output event */
		bpf_perf_event_output(ctx, &perf_map, BPF_F_CURRENT_CPU,
						perf_event, sizeof(struct perf_event));
	}
	ep->target.run_delay_ns = cur_task_run_delay_ns;

	return 0;
}

char LICENSE[] SEC("license") = "GPL";
