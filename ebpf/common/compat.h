/* Compatibility header for libbpf on systems where vmlinux.h types aren't recognized */
#ifndef GLASSHOUSE_COMPAT_H
#define GLASSHOUSE_COMPAT_H

/* Define kernel types if not already defined by vmlinux.h */
#ifndef __u8
typedef unsigned char __u8;
#endif

#ifndef __u16
typedef unsigned short __u16;
#endif

#ifndef __u32
typedef unsigned int __u32;
#endif

#ifndef __u64
typedef unsigned long long __u64;
#endif

#ifndef __s8
typedef signed char __s8;
#endif

#ifndef __s16
typedef signed short __s16;
#endif

#ifndef __s32
typedef signed int __s32;
#endif

#ifndef __s64
typedef signed long long __s64;
#endif

#endif /* GLASSHOUSE_COMPAT_H */

