#define BPF_NO_GLOBAL_DATA
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
 
typedef unsigned int u32;
typedef int pid_t;
const pid_t pid_filter = 0;
 
char LICENSE[] SEC("license") = "Dual BSD/GPL";

struct kill_event {
    __u32 pid;       // proceso que ejecutó kill()
    __u32 target;    // PID al que se envió la señal
    __s32 signal;    // SIGTERM, SIGKILL, etc.
}; 

SEC("tracepoint/syscalls/sys_enter_kill")
int trace_kill(struct trace_event_raw_sys_enter *ctx)
{
    struct kill_event event = {};
    event.pid = bpf_get_current_pid_tgid() >> 32;
    event.target = (u32)ctx->args[0];
    event.signal = (s32)ctx->args[1];

    bpf_printk("BPF triggered sys_enter_kill from PID %d to target PID %d with signal %d.\n", event.pid, event.target, event.signal);
    return 0;
}
