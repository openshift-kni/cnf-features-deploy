// SPDX-License-Identifier: GPL-2.0-or-later
#ifndef __RT_UTILS_H
#define __RT_UTILS_H

#include <assert.h>
#include <inttypes.h>
#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdarg.h>
#include <unistd.h>
#include <fcntl.h>
#include <getopt.h>
#include <pthread.h>
#include <signal.h>
#include <sched.h>
#include <string.h>
#include <time.h>
#include <errno.h>
#include <numa.h>
#include <math.h>
#include <limits.h>
#include <linux/unistd.h>

#include <sys/prctl.h>
#include <sys/stat.h>
#include <sys/sysinfo.h>
#include <sys/types.h>
#include <sys/time.h>
#include <sys/resource.h>
#include <sys/utsname.h>
#include <sys/mman.h>
#include <sys/syscall.h>

#include "error.h"

#define _STR(x) #x
#define STR(x) _STR(x)
#define MAX_PATH 256

int check_privs(void);
char *get_debugfileprefix(void);
int mount_debugfs(char *);
int get_tracers(char ***);
int valid_tracer(char *);

int setevent(char *event, char *val);
int event_enable(char *event);
int event_disable(char *event);
int event_enable_all(void);
int event_disable_all(void);

const char *policy_to_string(int policy);
uint32_t string_to_policy(const char *str);

/* trace.c */
void enable_trace_mark(void);
void tracemark(char *fmt, ...) __attribute__((format(printf, 1, 2)));

#endif	/* __RT_UTILS.H */
