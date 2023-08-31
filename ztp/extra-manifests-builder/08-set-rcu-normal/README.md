This generates a MachineConfig file that alters the state of rcu_normal/rcu_expedited. It will disable rcu_expedited by setting rcu_normal after the system has booted.

The script in question which does the work will exit after a given time window or, given appropriate configuration in environment variables, when the system reaches a "steady-state". This is defined as the number of running containers (according to crictl ps) not increasing or decreasing by a threshold factor over a time window. See the comments at the top of set-rcu-normal.sh for a more in-depth discussion of the variables that control this behavior.

The startup systemd job configures a maximum time of 10 minutes with a steady-state defined as an invariant count within 2% over 2 minutes once the total container count is greater than 40.
