# Guest Rootfs

Root filesystem build context for guest images.

- Include only what is required to execute workloads and run the guest probe.
- Avoid bundling secrets; prefer host-provided mounts for configuration.
- Keep kernel/userland versions compatible with CO-RE requirements to support optional guest profiling.
