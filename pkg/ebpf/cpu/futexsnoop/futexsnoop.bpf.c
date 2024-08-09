/*
 * Create: Fri Mar 29 23:05:25 2024
 */
#include <vmlinux.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_tracing.h>
#include <asm-generic/errno.h>
#include "bits.bpf.h"

#include "futexsnoop.h"

#define MAX_ENTRIES 10240

#define FUTEX_WAIT 0
#define FUTEX_PRIVATE_FLAG 128
#define FUTEX_CLOCK_REALTIME 256
#define FUTEX_CMD_MASK ~(FUTEX_PRIVATE_FLAG | FUTEX_CLOCK_REALTIME)

struct val_t {
	u64 ts;
	u64 uaddr;
};

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, MAX_ENTRIES);
	__type(key, u64);
	__type(value, struct val_t);
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
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, MAX_ENTRIES);
	__type(key, u64);
	__type(value, struct lock_stat);
} lock_map SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, MAX_ENTRIES);
	__type(key, struct hist_key);
	__type(value, struct hist);
} hists_map SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
	__uint(key_size, sizeof(u32));
	__uint(value_size, sizeof(u32));
} perf_map SEC(".maps");

static __always_inline void *
bpf_map_lookup_or_try_init(void *map, const void *key, const void *init)
{
	void *val;
	int err;

	val = bpf_map_lookup_elem(map, key);
	if (val)
		return val;

	err = bpf_map_update_elem(map, key, init, BPF_NOEXIST);
	if (err && err != -EEXIST)
		return 0;

	return bpf_map_lookup_elem(map, key);
}

SEC("tracepoint/syscalls/sys_enter_futex")
int futex_enter(struct syscall_trace_enter *ctx)
{
	u32 zero = 0;
	struct val_t v = {};
	u64 pid_tgid;
	struct lock_stat *lock = NULL;
	struct user_args *args = NULL;
	struct lock_stat lock_init = {
		.user_cnt = 1,
		.max_user_cnt = 1,
		.uaddr = 0,
	};
	u32 tid, pid;
	struct lock_stat event = { 0 };

	if (((int)ctx->args[1] & FUTEX_CMD_MASK) != FUTEX_WAIT)
		return 0;

	v.uaddr = ctx->args[0];
	v.ts = bpf_ktime_get_ns();

	args = bpf_map_lookup_elem(&args_map, &zero);
	if (!args)
		return 0;

	pid_tgid = bpf_get_current_pid_tgid();
	tid = (__u32)pid_tgid;
	pid = pid_tgid >> 32;

	if (args->targ_pid && args->targ_pid != pid)
		return 0;
	if (args->targ_tid && args->targ_tid != tid)
		return 0;
	if (args->targ_lock && args->targ_lock != v.uaddr)
		return 0;

	bpf_map_update_elem(&start, &pid_tgid, &v, BPF_ANY);

	lock = bpf_map_lookup_elem(&lock_map, &v.uaddr);
	if (lock) {
		__sync_fetch_and_add(&lock->user_cnt, 1);
		if (lock->user_cnt > lock->max_user_cnt) {
			lock->max_user_cnt = lock->user_cnt;
		}

		if (lock->max_user_cnt > args->max_lock_hold_users) {
			event = *lock;
			event.pid_tgid = pid_tgid;
			event.ts = v.ts;
			bpf_get_current_comm(event.comm, sizeof(event.comm));
			bpf_perf_event_output(ctx, &perf_map, BPF_F_CURRENT_CPU,
					      &event, sizeof(event));
		}
	} else {
		lock_init.uaddr = v.uaddr;
		bpf_map_update_elem(&lock_map, &v.uaddr, &lock_init, BPF_ANY);
	}

	return 0;
}

SEC("tracepoint/syscalls/sys_exit_futex")
int futex_exit(struct syscall_trace_exit *ctx)
{
	u64 pid_tgid, slot, ts, min, max, avg;
	struct hist_key hkey = {};
	struct hist *histp;
	struct val_t *vp;
	struct lock_stat *lock = NULL;
	struct hist initial_hist = {};
	struct user_args *args = NULL;
	u32 zero = 0;
	s64 delta;
	struct hist event = { 0 };

	ts = bpf_ktime_get_ns();
	pid_tgid = bpf_get_current_pid_tgid();
	vp = bpf_map_lookup_elem(&start, &pid_tgid);
	if (!vp)
		return 0;

	lock = bpf_map_lookup_elem(&lock_map, &vp->uaddr);
	if (lock)
		__sync_fetch_and_add(&lock->user_cnt, -1);

	if ((int)ctx->ret < 0)
		goto cleanup;

	delta = (s64)(ts - vp->ts);
	if (delta < 0)
		goto cleanup;

	hkey.pid_tgid = pid_tgid;
	hkey.uaddr = vp->uaddr;
	args = bpf_map_lookup_elem(&args_map, &zero);
	if (!args)
		return 0;

	if (args->stack)
		hkey.user_stack_id =
			bpf_get_stackid(ctx, &stack_map, BPF_F_USER_STACK);

	histp = bpf_map_lookup_or_try_init(&hists_map, &hkey, &initial_hist);
	if (!histp)
		goto cleanup;

	delta /= 1000000U;

	slot = log2l(delta);
	if (slot >= MAX_SLOTS)
		slot = MAX_SLOTS - 1;
	__sync_fetch_and_add(&histp->slots[slot], 1);
	__sync_fetch_and_add(&histp->contended, 1);
	__sync_fetch_and_add(&histp->total_elapsed, delta);
#ifdef __TARGET_ARCH_amd64
	min = __sync_fetch_and_add(&histp->min_dur, 0);
	if (!min || min > delta)
		__sync_val_compare_and_swap(&histp->min_dur, min, delta);
	max = __sync_fetch_and_add(&histp->max_dur, 0);
	if (max < delta) {
		__sync_val_compare_and_swap(&histp->max_dur, max, delta);
		histp->max_dur_ts = ts;
	}
#else
	/* arm64 not support automic instruction like: __sync_val_compare_and_swap in kernel-5.15 */
	min = histp->min_dur;
	if (!min || min > delta)
		histp->min_dur = delta;
	max = histp->max_dur;
	if (max < delta) {
		histp->max_dur = max;
		histp->max_dur_ts = ts;
	}
#endif
	avg = histp->total_elapsed / histp->contended;
	bpf_get_current_comm(&histp->comm, sizeof(histp->comm));
	if (delta > args->min_dur_ms && delta < args->max_dur_ms &&
	    delta > avg) {
		histp->uaddr = hkey.uaddr;
		histp->pid_tgid = hkey.pid_tgid;
		histp->user_stack_id = hkey.user_stack_id;
		histp->delta_dur = delta;
		event = *histp;
		bpf_perf_event_output(ctx, &perf_map, BPF_F_CURRENT_CPU, &event,
				      sizeof(event));
	}

cleanup:
	bpf_map_delete_elem(&start, &pid_tgid);
	return 0;
}

char LICENSE[] SEC("license") = "GPL";
