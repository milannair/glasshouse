#include "common.h"

static __always_inline void capture_cmdline(char *dst, int dst_len, const char *filename) {
	if (filename) {
		bpf_probe_read_user_str(dst, dst_len, filename);
	}
}

SEC("tracepoint/syscalls/sys_enter_execve")
int trace_execve(struct trace_event_raw_sys_enter *ctx) {
	struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e) {
		return 0;
	}
	__builtin_memset(e, 0, sizeof(*e));
	e->type = EVENT_EXEC;
	fill_common(e);

	const char *filename = (const char *)ctx->args[0];
	capture_cmdline(e->filename, sizeof(e->filename), filename);

	bpf_ringbuf_submit(e, 0);
	return 0;
}

SEC("tracepoint/syscalls/sys_enter_execveat")
int trace_execveat(struct trace_event_raw_sys_enter *ctx) {
	struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e) {
		return 0;
	}
	__builtin_memset(e, 0, sizeof(*e));
	e->type = EVENT_EXEC;
	fill_common(e);

	const char *filename = (const char *)ctx->args[1];
	capture_cmdline(e->filename, sizeof(e->filename), filename);

	bpf_ringbuf_submit(e, 0);
	return 0;
}

char LICENSE[] SEC("license") = "Dual BSD/GPL";
