#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/init.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/sched.h>
#include <linux/sched/signal.h>
#include <linux/mm.h>
#include <linux/slab.h>
#include <linux/uaccess.h>
#include <linux/fs.h>
#include <linux/string.h>
#include <linux/nsproxy.h>
#include <linux/pid_namespace.h>
#include <linux/cgroup.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Estudiante USAC");
MODULE_DESCRIPTION("Modulo de kernel para telemetria de contenedores - SO1");
MODULE_VERSION("1.0");


#define PROC_FILENAME "continfo_pr1_so1_201800632"

static struct proc_dir_entry *proc_entry;

/*
 * Obtiene el porcentaje de CPU de un proceso usando utime + stime.
 * El kernel devuelve valores en jiffies; lo convertimos a un entero
 * multiplicado x100 para evitar flotantes (sin soporte en kernel).
 * Nota: como el spec indica, valores grandes son aceptables.
 */
static unsigned long get_cpu_percent(struct task_struct *task)
{
    unsigned long utime = 0, stime = 0;
    unsigned long total_jiffies;

    utime = task->utime;
    stime = task->stime;

    total_jiffies = utime + stime;

    if (HZ == 0)
        return 0;

    return (total_jiffies * 100) / HZ;
}

static int is_container_process(struct task_struct *task)
{
    struct task_struct *parent;
    char comm[TASK_COMM_LEN];

    /* Nombres de procesos típicos de Docker */
    const char *container_procs[] = {
        "containerd", "dockerd", "docker", "runc",
        "docker-proxy", "containerd-shim", NULL
    };
    int i;

    get_task_comm(comm, task);

    for (i = 0; container_procs[i] != NULL; i++) {
        if (strstr(comm, container_procs[i]) != NULL)
            return 1;
    }

    /* Revisar si algún ancestro es un proceso de contenedor */
    parent = task->parent;
    while (parent && parent->pid > 1) {
        get_task_comm(comm, parent);
        for (i = 0; container_procs[i] != NULL; i++) {
            if (strstr(comm, container_procs[i]) != NULL)
                return 1;
        }
        parent = parent->parent;
    }

    return 0;
}


// Función principal: genera el contenido del archivo /proc
static int continfo_show(struct seq_file *m, void *v)
{
    struct task_struct *task;
    struct mm_struct *mm;

    // Métricas de RAM del sistema 
    struct sysinfo si;
    si_meminfo(&si);

    unsigned long total_ram_mb  = (si.totalram * si.mem_unit) >> 20;
    unsigned long free_ram_mb   = (si.freeram  * si.mem_unit) >> 20;
    unsigned long used_ram_mb   = total_ram_mb - free_ram_mb;

    seq_printf(m, "{\n");
    seq_printf(m, "  \"ram_total_mb\": %lu,\n", total_ram_mb);
    seq_printf(m, "  \"ram_free_mb\": %lu,\n",  free_ram_mb);
    seq_printf(m, "  \"ram_used_mb\": %lu,\n",  used_ram_mb);
    seq_printf(m, "  \"processes\": [\n");

    // Iteración sobre todos los procesos
    int first = 1;

    rcu_read_lock();
    for_each_process(task) {
        char comm[TASK_COMM_LEN];
        unsigned long vsz_kb = 0;
        unsigned long rss_kb = 0;
        unsigned long mem_percent = 0;
        unsigned long cpu_percent = 0;
        char cmdline[256];
        int is_container;

        // Solo procesos activos
        if (task->pid <= 0)
            continue;

        get_task_comm(comm, task);
        is_container = is_container_process(task);

        // Memoria virtual (VSZ) y física (RSS)
        mm = task->mm;
        if (mm) {
            vsz_kb = (mm->total_vm << PAGE_SHIFT) >> 10;
            rss_kb = (get_mm_rss(mm) << PAGE_SHIFT) >> 10;

            /* Porcentaje de RAM: (RSS / total_ram) * 100 */
            if (si.totalram > 0) {
                mem_percent = (rss_kb * 100) /
                              ((si.totalram * si.mem_unit) >> 10);
            }
        }

        cpu_percent = get_cpu_percent(task);

        snprintf(cmdline, sizeof(cmdline), "%s[%d]", comm, task->pid);

        if (!first)
            seq_printf(m, ",\n");
        first = 0;

        seq_printf(m, "    {\n");
        seq_printf(m, "      \"pid\": %d,\n",           task->pid);
        seq_printf(m, "      \"name\": \"%s\",\n",      comm);
        seq_printf(m, "      \"cmdline\": \"%s\",\n",   cmdline);
        seq_printf(m, "      \"vsz_kb\": %lu,\n",       vsz_kb);
        seq_printf(m, "      \"rss_kb\": %lu,\n",       rss_kb);
        seq_printf(m, "      \"mem_percent\": %lu,\n",  mem_percent);
        seq_printf(m, "      \"cpu_percent\": %lu,\n",  cpu_percent);
        seq_printf(m, "      \"is_container\": %d\n",   is_container);
        seq_printf(m, "    }");
    }
    rcu_read_unlock();

    seq_printf(m, "\n  ]\n");
    seq_printf(m, "}\n");

    return 0;
}

static int continfo_open(struct inode *inode, struct file *file)
{
    return single_open(file, continfo_show, NULL);
}

/* Operaciones del archivo /proc — compatibles con kernels modernos */
static const struct proc_ops continfo_fops = {
    .proc_open    = continfo_open,
    .proc_read    = seq_read,
    .proc_lseek   = seq_lseek,
    .proc_release = single_release,
};


// Init / Exit/
static int __init continfo_init(void)
{
    proc_entry = proc_create(PROC_FILENAME, 0444, NULL, &continfo_fops);
    if (!proc_entry) {
        pr_err("continfo: No se pudo crear /proc/%s\n", PROC_FILENAME);
        return -ENOMEM;
    }
    pr_info("continfo: modulo cargado -> /proc/%s\n", PROC_FILENAME);
    return 0;
}

static void __exit continfo_exit(void)
{
    proc_remove(proc_entry);
    pr_info("continfo: modulo descargado\n");
}

module_init(continfo_init);
module_exit(continfo_exit);
