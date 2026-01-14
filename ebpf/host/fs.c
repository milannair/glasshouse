#include "common.h"

SEC("tracepoint/syscalls/sys_enter_openat")
int trace_openat(struct trace_event_raw_sys_enter *ctx) {
	struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e) {
		return 0;
	}
	__builtin_memset(e, 0, sizeof(*e));
	e->type = EVENT_OPEN;
	fill_common(e);

	const char *filename = (const char *)ctx->args[1];
	e->flags = (__u32)ctx->args[2];
	bpf_probe_read_user_str(e->filename, sizeof(e->filename), filename);

	bpf_ringbuf_submit(e, 0);
	return 0;
}

SEC("tracepoint/syscalls/sys_enter_open")
int trace_open(struct trace_event_raw_sys_enter *ctx) {
	struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e) {
		return 0;
	}
	__builtin_memset(e, 0, sizeof(*e));
	e->type = EVENT_OPEN;
	fill_common(e);

	const char *filename = (const char *)ctx->args[0];
	e->flags = (__u32)ctx->args[1];
	bpf_probe_read_user_str(e->filename, sizeof(e->filename), filename);

	bpf_ringbuf_submit(e, 0);
	return 0;
}

char LICENSE[] SEC("license") = "Dual BSD/GPL";
