This generates a MachineConfig file that alters the CPU pinning on a machine
with workload partitioning enabled. It moves a set of critical system processes
off of the restricted CPUs and to the full complement of available CPUs during
startup and shutdown.

The script in question which does the work will exit after a given time window
or, given appropriate configuration in environment variables, when the system
reaches a "steady-state". This is defined as the number of running containers
(according to `crictl ps`) not increasing or decreasing by a threshold factor
over a time window.  See the comments at the top of
[accelerated-container-startup.sh](accelerated-container-startup.sh) for a more
in-depth discussion of the variables that control this behavior.

The [startup systemd job](accelerated-container-startup.service) configures a
maximum time of 10 minutes with a steady-state defined as an invariant count
within 2% over 2 minutes once the total container count is greater than 40.

The [shutdown systemd job](accelerated-container-shutdown.service) configures a
maximum time of 10 minutes and disable the steady-state check.
