
#define PERF_MAX_STACK_DEPTH 127
#define PROFILE_MAPS_SIZE 16384

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
