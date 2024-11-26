/*
 * Copyright: Copyright (c) Tao Chen
 * Author: Tao Chen <chen.dylane@gmail.com>
 */

#include <vmlinux.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_tracing.h>

/* https://github.com/iovisor/bcc/issues/237 */

#include "cachestat.h"

#define CACHESTAT_MAX_ENTRIES CACHESTAT_MAP_SIZE

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__type(key, u32);
	__type(value, struct user_args);
	__uint(max_entries, 8);
} args_map SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__type(key, struct cache_key);
	__type(value, struct cache_info);
	__uint(max_entries, CACHESTAT_MAX_ENTRIES);
} cachestat_map SEC(".maps");

SEC("kprobe/vfs_write")
int vfs_write_hook(struct pt_regs *ctx)
{
	struct cache_key key = { 0 };
	struct cache_info cache_info = { 0 };
	struct cache_info *info = NULL;
	struct file *f = NULL;
	struct dentry *de = NULL;
	struct qstr dn = { 0 };
	struct user_args *args = NULL;
	int count, index = 0;
	u64 pid_tgid = bpf_get_current_pid_tgid();
	key.pid = pid_tgid >> 32;
	args = bpf_map_lookup_elem(&args_map, &index);
	if (!args)
		return 0;
	if (args->pid != -1 && args->pid != key.pid) {
		return 0;
	}
	if (args->cache_type == READ_CACHE)
		return 0;

	f = (struct file *)PT_REGS_PARM1_CORE(ctx);
	de = (struct dentry *)BPF_CORE_READ(f, f_path.dentry);
	dn = BPF_CORE_READ(de, d_name);
	count = (int)PT_REGS_PARM3_CORE(ctx);

	key.file = dn.name;
	info = bpf_map_lookup_elem(&cachestat_map, &key);
	if (!info) {
		cache_info.pid = key.pid;
		bpf_get_current_comm(cache_info.comm, sizeof(cache_info.comm));
		if (dn.name != NULL)
			bpf_core_read(&cache_info.file, sizeof(cache_info.file),
				      dn.name);
		else
			cache_info.file[0] = '?';
		cache_info.write_size = count;
		bpf_map_update_elem(&cachestat_map, &key, &cache_info, 0);
		return 0;
	}
	info->write_size = info->write_size + count;
	return 0;
}

SEC("kprobe/vfs_read")
int vfs_read_hook(struct pt_regs *ctx)
{
	struct cache_key key = { 0 };
	struct cache_info cache_info = { 0 };
	struct cache_info *info = NULL;
	struct file *f = NULL;
	struct dentry *de = NULL;
	struct qstr dn = { 0 };
	struct user_args *args = NULL;
	int count, index = 0;
	u64 pid_tgid = bpf_get_current_pid_tgid();
	key.pid = pid_tgid >> 32;

	args = bpf_map_lookup_elem(&args_map, &index);
	if (!args)
		return 0;
	if (args->pid != -1 && args->pid != key.pid)
		return 0;
	if (args->cache_type == WRITE_CACHE)
		return 0;

	f = (struct file *)PT_REGS_PARM1_CORE(ctx);
	de = (struct dentry *)BPF_CORE_READ(f, f_path.dentry);
	dn = BPF_CORE_READ(de, d_name);
	count = (int)PT_REGS_PARM3_CORE(ctx);

	key.file = dn.name;
	info = bpf_map_lookup_elem(&cachestat_map, &key);
	if (!info) {
		cache_info.pid = key.pid;
		bpf_get_current_comm(cache_info.comm, sizeof(cache_info.comm));
		if (dn.name != NULL)
			bpf_core_read(&cache_info.file, sizeof(cache_info.file),
				      dn.name);
		else
			cache_info.file[0] = '?';
		cache_info.read_size = count;
		bpf_map_update_elem(&cachestat_map, &key, &cache_info, 0);
		return 0;
	}
	info->read_size = info->read_size + count;

	return 0;
}

char LICENSE[] SEC("license") = "GPL";