#ifndef GLASSHOUSE_COMMON_H
#define GLASSHOUSE_COMMON_H

/* Include compat.h first for type definitions needed by libbpf headers */
#include "compat.h"
#include "vmlinux.h"
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

#define PATH_MAX 256
#define COMM_MAX 16
#define ADDR_LEN 16

enum event_type {
	EVENT_EXEC = 1,
	EVENT_OPEN = 2,
	EVENT_CONNECT = 3,
};

struct event {
	__u32 type;
	__u32 pid;
	__u32 ppid;
	__u32 flags;
	__u16 port;
	__u8 addr_family;
	__u8 proto;
	__u8 addr[ADDR_LEN];
	char comm[COMM_MAX];
	char filename[PATH_MAX];
};

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 24);
} events SEC(".maps");

static __always_inline __u32 get_ppid(void) {
	struct task_struct *task = (struct task_struct *)bpf_get_current_task();
	return BPF_CORE_READ(task, real_parent, tgid);
}

static __always_inline void fill_common(struct event *e) {
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	e->pid = pid_tgid >> 32;
	e->ppid = get_ppid();
	bpf_get_current_comm(&e->comm, sizeof(e->comm));
}

#endif
