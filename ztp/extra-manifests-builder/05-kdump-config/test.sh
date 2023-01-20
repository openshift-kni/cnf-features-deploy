#!/bin/bash

fatal() {
    echo "FATAL: $@"
    exit 1
}

export TMPDIR=$(mktemp -d)

cleanup() {
    rm -rf $TMPDIR
}
trap cleanup EXIT

cat >$TMPDIR/kdump.good <<EOF
# Kernel Version string for the -kdump kernel, such as 2.6.13-1544.FC5kdump
# If no version is specified, then the init script will try to find a
# kdump kernel with the same version number as the running kernel.
KDUMP_KERNELVER=""

# The kdump commandline is the command line that needs to be passed off to
# the kdump kernel.  This will likely match the contents of the grub kernel
# line.  For example:
#   KDUMP_COMMANDLINE="ro root=LABEL=/"
# Dracut depends on proper root= options, so please make sure that appropriate
# root= options are copied from /proc/cmdline. In general it is best to append
# command line options using "KDUMP_COMMANDLINE_APPEND=".
# If a command line is not specified, the default will be taken from
# /proc/cmdline
KDUMP_COMMANDLINE=""

# This variable lets us remove arguments from the current kdump commandline
# as taken from either KDUMP_COMMANDLINE above, or from /proc/cmdline
# NOTE: some arguments such as crashkernel will always be removed
KDUMP_COMMANDLINE_REMOVE="ignition.firstboot hugepages hugepagesz slub_debug quiet log_buf_len swiotlb"

# This variable lets us append arguments to the current kdump commandline
# after processed by KDUMP_COMMANDLINE_REMOVE
KDUMP_COMMANDLINE_APPEND="irqpoll nr_cpus=1 reset_devices cgroup_disable=memory mce=off numa=off udev.children-max=2 panic=10 rootflags=nofail acpi_no_memhotplug transparent_hugepage=never nokaslr novmcoredd hest_disable"

# Any additional kexec arguments required.  In most situations, this should
# be left empty
#
# Example:
#   KEXEC_ARGS="--elf32-core-headers"
KEXEC_ARGS="-s"
EOF

cat >$TMPDIR/kdump.good.expected <<EOF
# Kernel Version string for the -kdump kernel, such as 2.6.13-1544.FC5kdump
# If no version is specified, then the init script will try to find a
# kdump kernel with the same version number as the running kernel.
KDUMP_KERNELVER=""

# The kdump commandline is the command line that needs to be passed off to
# the kdump kernel.  This will likely match the contents of the grub kernel
# line.  For example:
#   KDUMP_COMMANDLINE="ro root=LABEL=/"
# Dracut depends on proper root= options, so please make sure that appropriate
# root= options are copied from /proc/cmdline. In general it is best to append
# command line options using "KDUMP_COMMANDLINE_APPEND=".
# If a command line is not specified, the default will be taken from
# /proc/cmdline
KDUMP_COMMANDLINE=""

# This variable lets us remove arguments from the current kdump commandline
# as taken from either KDUMP_COMMANDLINE above, or from /proc/cmdline
# NOTE: some arguments such as crashkernel will always be removed
KDUMP_COMMANDLINE_REMOVE="ignition.firstboot hugepages hugepagesz slub_debug quiet log_buf_len swiotlb"

# This variable lets us append arguments to the current kdump commandline
# after processed by KDUMP_COMMANDLINE_REMOVE
KDUMP_COMMANDLINE_APPEND="irqpoll nr_cpus=1 reset_devices cgroup_disable=memory mce=off numa=off udev.children-max=2 panic=10 rootflags=nofail acpi_no_memhotplug transparent_hugepage=never nokaslr novmcoredd hest_disable module_blacklist=ice"

# Any additional kexec arguments required.  In most situations, this should
# be left empty
#
# Example:
#   KEXEC_ARGS="--elf32-core-headers"
KEXEC_ARGS="-s"
EOF

cat >$TMPDIR/kdump.no_option <<EOF
# Simple test file that does not contain the KDUMP_COMMANDLINE_APPEND option
KDUMP_KERNELVER=""
EOF

echo "Test: File modified"
cp $TMPDIR/kdump.good $TMPDIR/kdump
bash ./kdump-remove-ice-module.sh $TMPDIR/kdump
[[ $? -eq 0 ]] || fatal "kdump-remove-ice-module.sh unexpected exit code $?"
diff $TMPDIR/kdump $TMPDIR/kdump.good.expected
[[ $? -eq 0 ]] || fatal "kdump file was not modified as expected"
rm $TMPDIR/kdump

echo "Test: File not modified on second run"
cp $TMPDIR/kdump.good $TMPDIR/kdump
bash ./kdump-remove-ice-module.sh $TMPDIR/kdump
[[ $? -eq 0 ]] || fatal "kdump-remove-ice-module.sh unexpected exit code $?"
diff $TMPDIR/kdump $TMPDIR/kdump.good.expected
[[ $? -eq 0 ]] || fatal "kdump file was not modified as expected"
bash ./kdump-remove-ice-module.sh $TMPDIR/kdump
[[ $? -eq 0 ]] || fatal "kdump-remove-ice-module.sh unexpected exit code $?"
diff $TMPDIR/kdump $TMPDIR/kdump.good.expected
[[ $? -eq 0 ]] || fatal "kdump file was modified on second run"
rm $TMPDIR/kdump

echo "Test: File not modified if option missing"
cp $TMPDIR/kdump.no_option $TMPDIR/kdump
bash ./kdump-remove-ice-module.sh $TMPDIR/kdump
[[ $? -eq 0 ]] || fatal "kdump-remove-ice-module.sh unexpected exit code $?"
diff $TMPDIR/kdump $TMPDIR/kdump.no_option
[[ $? -eq 0 ]] || fatal "kdump file was unexpectedly modified"
rm $TMPDIR/kdump

echo "Test: File doesn't exist"
bash ./kdump-remove-ice-module.sh $TMPDIR/missing-kdump-file
[[ $? -eq 0 ]] || fatal "kdump-remove-ice-module.sh unexpected exit code $?"

echo "All test completed successfully"
