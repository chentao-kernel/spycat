/*
 * Create: Tue May 21 15:18:47 2024
 */
#ifndef __SYSCALL_H
#define __SYSCALL_H

struct user_args {
	__u32 pid;
	__u32 tid;
	__u32 min_syscall_ms;
	__u32 max_syscall_ms;
	bool stack;
	__u32 syscall_id;
};

struct syscall_event {
	u32 pid;
	u32 tid;
	u64 ts_ns;
	u64 dur_us;
	s64 u_stack_id;
	s64 k_stack_id;
	u64 syscall_id;
	char comm[16];
};

#endif /* __SYSCALL_H */
