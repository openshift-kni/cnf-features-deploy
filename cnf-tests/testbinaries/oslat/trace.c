// SPDX-License-Identifier: GPL-2.0-only
/*
 * This is part of cyclictest.c from rt-test:
 * git://git.kernel.org/pub/scm/utils/rt-tests/rt-tests.git
 *
 * (C) 2013      Clark Williams <williams@redhat.com>
 * (C) 2013      John Kacur <jkacur@redhat.com>
 * (C) 2008-2012 Clark Williams <williams@redhat.com>
 * (C) 2005-2007 Thomas Gleixner <tglx@linutronix.de>
 *
 */

#include "rt-utils.h"

static char *fileprefix;
static int trace_fd     = -1;
static int tracemark_fd = -1;

static void open_tracemark_fd(void)
{
	char path[MAX_PATH];

	/*
	 * open the tracemark file if it's not already open
	 */
	if (tracemark_fd < 0) {
		sprintf(path, "%s/%s", fileprefix, "trace_marker");
		tracemark_fd = open(path, O_WRONLY);
		if (tracemark_fd < 0) {
			warn("unable to open trace_marker file: %s\n", path);
			return;
		}
	}

	/*
	 * if we're not tracing and the tracing_on fd is not open,
	 * open the tracing_on file so that we can stop the trace
	 * if we hit a breaktrace threshold
	 */
	if (trace_fd < 0) {
		sprintf(path, "%s/%s", fileprefix, "tracing_on");
		if ((trace_fd = open(path, O_WRONLY)) < 0)
			warn("unable to open tracing_on file: %s\n", path);
	}
}

static int trace_file_exists(char *name)
{
       struct stat sbuf;
       char *tracing_prefix = get_debugfileprefix();
       char path[MAX_PATH];
       strcat(strcpy(path, tracing_prefix), name);
       return stat(path, &sbuf) ? 0 : 1;
}

static void debugfs_prepare(void)
{
	if (mount_debugfs(NULL))
		fatal("could not mount debugfs");

	fileprefix = get_debugfileprefix();
	if (!trace_file_exists("tracing_enabled") &&
	    !trace_file_exists("tracing_on"))
		warn("tracing_enabled or tracing_on not found\n"
		     "debug fs not mounted");
}

#define TRACEBUFSIZ 1024
static __thread char tracebuf[TRACEBUFSIZ];

void tracemark(char *fmt, ...)
{
	va_list ap;
	int len;

	/* bail out if we're not tracing */
	/* or if the kernel doesn't support trace_mark */
	if (tracemark_fd < 0 || trace_fd < 0)
		return;

	va_start(ap, fmt);
	len = vsnprintf(tracebuf, TRACEBUFSIZ, fmt, ap);
	va_end(ap);

	/* write the tracemark message */
	write(tracemark_fd, tracebuf, len);

	/* now stop any trace */
	write(trace_fd, "0\n", 2);
}

void enable_trace_mark(void)
{
	debugfs_prepare();
	open_tracemark_fd();
}
