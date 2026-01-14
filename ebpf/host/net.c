#include "common.h"

// Define socket and protocol constants (structures are in vmlinux.h)
#define AF_INET  2
#define AF_INET6 10

#define SOCK_STREAM 1
#define SOCK_DGRAM  2
#define SOCK_TYPE_MASK 0xf

#define IPPROTO_TCP 6
#define IPPROTO_UDP 17

struct socket_args {
	__u32 domain;
	__u32 type;
	__u32 protocol;
};

struct socket_key {
	__u32 pid;
	__u32 fd;
};

struct socket_meta {
	__u8 protocol;
	__u8 pad[3];
};

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 8192);
	__type(key, __u32);
	__type(value, struct socket_args);
} socket_args_map SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, 16384);
	__type(key, struct socket_key);
	__type(value, struct socket_meta);
} socket_meta_map SEC(".maps");

static __always_inline __u8 infer_proto(struct socket_args *args) {
	if (!args) {
		return 0;
	}
	if (args->protocol != 0) {
		return (__u8)args->protocol;
	}
	__u32 type = args->type & SOCK_TYPE_MASK;
	if (type == SOCK_STREAM) {
		return IPPROTO_TCP;
	}
	if (type == SOCK_DGRAM) {
		return IPPROTO_UDP;
	}
	return 0;
}

SEC("tracepoint/syscalls/sys_enter_socket")
int trace_socket_enter(struct trace_event_raw_sys_enter *ctx) {
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 pid = pid_tgid >> 32;
	struct socket_args args = {};

	args.domain = (__u32)ctx->args[0];
	args.type = (__u32)ctx->args[1];
	args.protocol = (__u32)ctx->args[2];

	bpf_map_update_elem(&socket_args_map, &pid, &args, BPF_ANY);
	return 0;
}

SEC("tracepoint/syscalls/sys_exit_socket")
int trace_socket_exit(struct trace_event_raw_sys_exit *ctx) {
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 pid = pid_tgid >> 32;
	struct socket_args *args = bpf_map_lookup_elem(&socket_args_map, &pid);
	if (!args) {
		return 0;
	}

	int fd = (int)ctx->ret;
	if (fd < 0) {
		bpf_map_delete_elem(&socket_args_map, &pid);
		return 0;
	}

	struct socket_key key = {
		.pid = pid,
		.fd = (__u32)fd,
	};
	struct socket_meta meta = {};
	meta.protocol = infer_proto(args);

	bpf_map_update_elem(&socket_meta_map, &key, &meta, BPF_ANY);
	bpf_map_delete_elem(&socket_args_map, &pid);
	return 0;
}

SEC("tracepoint/syscalls/sys_enter_connect")
int trace_connect(struct trace_event_raw_sys_enter *ctx) {
	struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e) {
		return 0;
	}
	__builtin_memset(e, 0, sizeof(*e));
	e->type = EVENT_CONNECT;
	fill_common(e);

	__u32 fd = (__u32)ctx->args[0];
	struct socket_key key = {
		.pid = e->pid,
		.fd = fd,
	};
	struct socket_meta *meta = bpf_map_lookup_elem(&socket_meta_map, &key);
	if (meta) {
		e->proto = meta->protocol;
	}

	struct sockaddr *addr = (struct sockaddr *)ctx->args[1];
	struct sockaddr_in sa4 = {};
	struct sockaddr_in6 sa6 = {};

	if (bpf_probe_read_user(&sa4, sizeof(sa4), addr) == 0) {
		if (sa4.sin_family == AF_INET) {
			e->addr_family = AF_INET;
			e->port = bpf_ntohs(sa4.sin_port);
			__builtin_memcpy(e->addr, &sa4.sin_addr, sizeof(sa4.sin_addr));
			bpf_ringbuf_submit(e, 0);
			return 0;
		}
	}

	if (bpf_probe_read_user(&sa6, sizeof(sa6), addr) == 0) {
		if (sa6.sin6_family == AF_INET6) {
			e->addr_family = AF_INET6;
			e->port = bpf_ntohs(sa6.sin6_port);
			__builtin_memcpy(e->addr, &sa6.sin6_addr, sizeof(sa6.sin6_addr));
			bpf_ringbuf_submit(e, 0);
			return 0;
		}
	}

	bpf_ringbuf_discard(e, 0);
	return 0;
}

char LICENSE[] SEC("license") = "Dual BSD/GPL";
