#include <linux/module.h>
#include <linux/export-internal.h>
#include <linux/compiler.h>

MODULE_INFO(name, KBUILD_MODNAME);

__visible struct module __this_module
__section(".gnu.linkonce.this_module") = {
	.name = KBUILD_MODNAME,
	.init = init_module,
#ifdef CONFIG_MODULE_UNLOAD
	.exit = cleanup_module,
#endif
	.arch = MODULE_ARCH_INIT,
};



static const struct modversion_info ____versions[]
__used __section("__versions") = {
	{ 0xc714db12, "single_open" },
	{ 0xbd03ed67, "__ref_stack_chk_guard" },
	{ 0xc7ffe1aa, "si_meminfo" },
	{ 0x5fe49b2b, "seq_printf" },
	{ 0xd272d446, "__rcu_read_lock" },
	{ 0x3bc51ecc, "init_task" },
	{ 0x9479a1e8, "strnlen" },
	{ 0xd70733be, "sized_strscpy" },
	{ 0x17545440, "strstr" },
	{ 0x40a621c5, "snprintf" },
	{ 0xd272d446, "__rcu_read_unlock" },
	{ 0x90a48d82, "__ubsan_handle_out_of_bounds" },
	{ 0xd272d446, "__stack_chk_fail" },
	{ 0xe54e0a6b, "__fortify_panic" },
	{ 0x51357216, "proc_remove" },
	{ 0xe290afe8, "seq_read" },
	{ 0xce9b2870, "seq_lseek" },
	{ 0x6f7437b5, "single_release" },
	{ 0xd272d446, "__fentry__" },
	{ 0xdef8136e, "proc_create" },
	{ 0xe8213e80, "_printk" },
	{ 0xd272d446, "__x86_return_thunk" },
	{ 0xe9196a28, "module_layout" },
};

static const u32 ____version_ext_crcs[]
__used __section("__version_ext_crcs") = {
	0xc714db12,
	0xbd03ed67,
	0xc7ffe1aa,
	0x5fe49b2b,
	0xd272d446,
	0x3bc51ecc,
	0x9479a1e8,
	0xd70733be,
	0x17545440,
	0x40a621c5,
	0xd272d446,
	0x90a48d82,
	0xd272d446,
	0xe54e0a6b,
	0x51357216,
	0xe290afe8,
	0xce9b2870,
	0x6f7437b5,
	0xd272d446,
	0xdef8136e,
	0xe8213e80,
	0xd272d446,
	0xe9196a28,
};
static const char ____version_ext_names[]
__used __section("__version_ext_names") =
	"single_open\0"
	"__ref_stack_chk_guard\0"
	"si_meminfo\0"
	"seq_printf\0"
	"__rcu_read_lock\0"
	"init_task\0"
	"strnlen\0"
	"sized_strscpy\0"
	"strstr\0"
	"snprintf\0"
	"__rcu_read_unlock\0"
	"__ubsan_handle_out_of_bounds\0"
	"__stack_chk_fail\0"
	"__fortify_panic\0"
	"proc_remove\0"
	"seq_read\0"
	"seq_lseek\0"
	"single_release\0"
	"__fentry__\0"
	"proc_create\0"
	"_printk\0"
	"__x86_return_thunk\0"
	"module_layout\0"
;

MODULE_INFO(depends, "");


MODULE_INFO(srcversion, "7E73C1E7CEC2689501D52F0");
