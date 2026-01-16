#include "common.h"

#define ARGS_MAX 8

static __always_inline int append_arg(char *dst, int dst_len, const char *src, int off) {
	if (!src) {
		return off;
	}
	if (off >= dst_len - 1) {
		return off;
	}
	if (off > 0) {
		dst[off] = ' ';
		off++;
		if (off >= dst_len - 1) {
			return off;
		}
	}
	int copied = bpf_probe_read_user_str(dst + off, dst_len - off, src);
	if (copied <= 0) {
		return off;
	}
	off += copied - 1;
	return off;
}

static __always_inline void capture_cmdline(char *dst, int dst_len, const char *filename,
					    const char *const *argv) {
	int off = 0;

#pragma unroll
	for (int i = 0; i < ARGS_MAX; i++) {
		const char *argp = NULL;
		if (bpf_probe_read_user(&argp, sizeof(argp), &argv[i]) < 0) {
			break;
		}
		if (!argp) {
			break;
		}
		off = append_arg(dst, dst_len, argp, off);
	}

	if (off == 0 && filename) {
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
	const char *const *argv = (const char *const *)ctx->args[1];
	capture_cmdline(e->filename, sizeof(e->filename), filename, argv);

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
	const char *const *argv = (const char *const *)ctx->args[2];
	capture_cmdline(e->filename, sizeof(e->filename), filename, argv);

	bpf_ringbuf_submit(e, 0);
	return 0;
}

char LICENSE[] SEC("license") = "Dual BSD/GPL";
