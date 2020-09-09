oslat - OS Latency Detector
==========

Introduction
------------

This is a test program for detecting OS level thread latency caused by
unexpected system scheduling or interruptions (e.g., system ticks).

Please run this program using root, or make sure you have the required
privileges for e.g. setting schedule priorities or doing memory locks.

Major features:

  - Poll-based busy loop, doing RDTSC on specified cores.  By default, it'll
    launch one test thread on each of the core.
  - Collect interruptions of rdtsc sequence (in microseconds) and put into
    per-us baskets (1us, 2us, 3us, ..., Nus, N configurable).
  - Supports CPU-list (libnuma), FIFO priority, and multi-threading
  - Little memory footprint
  - Supports ftrace

Sample output
-------------

This is a sample output of running oslat on a real-time virtual machine (core
2-9 isolated) for 1 hour:

    [root@localhost ~]# ./oslat --cpu-list 2-9 --rtprio 1 --runtime 3600

        Version: v0.1.0

        core_i:    2 3 4 5 6 7 8 9
        cpu_mhz:    2197 2197 2197 2197 2197 2197 2197 2197
        001 (us):    93068541026 93068541249 93068541429 93068541411 93068541565 93068541296 93068541320 93068541409
        002 (us):    51 51 51 51 51 51 51 51
        003 (us):    2 2 2 2 2 2 2 2
        004 (us):    4 4 8 3 26 3 4 2
        005 (us):    40 39 35 41 18 42 40 41
        006 (us):    1 1 1 0 0 0 1 1
        007 (us):    0 0 0 0 0 0 0 0
        008 (us):    0 0 0 0 0 0 0 0
        009 (us):    0 0 0 0 0 0 0 0
        010 (us):    0 0 0 0 0 0 0 0
        011 (us):    0 0 0 0 0 0 0 0
        012 (us):    0 0 0 0 0 0 0 0
        013 (us):    0 0 0 0 0 0 0 0
        014 (us):    0 0 0 0 0 0 0 0
        015 (us):    0 0 0 0 0 0 0 0
        016 (us):    0 0 0 0 0 0 0 0
        017 (us):    0 0 0 0 0 0 0 0
        018 (us):    0 0 0 0 0 0 0 0
        019 (us):    0 0 0 0 0 0 0 0
        020 (us):    0 0 0 0 0 0 0 0
        021 (us):    0 0 0 0 0 0 0 0
        022 (us):    0 0 0 0 0 0 0 0
        023 (us):    0 0 0 0 0 0 0 0
        024 (us):    0 0 0 0 0 0 0 0
        025 (us):    0 0 0 0 0 0 0 0
        026 (us):    0 0 0 0 0 0 0 0
        027 (us):    0 0 0 0 0 0 0 0
        028 (us):    0 0 0 0 0 0 0 0
        029 (us):    0 0 0 0 0 0 0 0
        030 (us):    0 0 0 0 0 0 0 0
        031 (us):    0 0 0 0 0 0 0 0
        032 (us):    0 0 0 0 0 0 0 0
        maxlat:    6 6 6 5 5 5 6 6 (us)
        runtime:    3600.740 3600.740 3600.740 3600.740 3600.740 3600.740 3600.740 3600.740 (sec)
