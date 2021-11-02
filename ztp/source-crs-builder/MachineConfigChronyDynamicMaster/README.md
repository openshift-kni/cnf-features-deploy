# README #
This generates a MachineConfig file for dynamic chronyd operation based on PTP 
synchronization state.

## Background ##
DU nodes are synchronized to a high precision time source using PTP protocol. 
PTP protocol is implemented by a set of daemons deployed and controlled by the Openshift PTP operator on the designated nodes. PTP operator is installed on the cluster after it has been deployed (“Day 2”).
All Openshift nodes are deployed by default with the “chronyd” daemon, which provides NTP time synchronization. 
Node time synchronization is essential to facilitate TLS handshakes, timestamping of fault / performance information, events and logs. 
NTP and PTP don’t work together smoothly, since they both update the same system time reference. If operated together, intermittent jumps occur to the system time.
There are solutions that allow PTP and NTP to [work together](https://www.redhat.com/en/blog/combining-ptp-ntp-get-best-both-worlds), but they haven't been verified to work with the Telco PTP profile and provide the performance required for DU workloads.
To overcome PTP-NTP interoperability issue, current practice is to disable NTP daemon (chronyd) on the nodes that PTP daemon is deployed to. This workaround has a number of caveats:
- Chronyd is disabled on “Day 2” using "Machineconfig object. This implies the node reboot - not acceptable from the deployment time perspective.
- The chronyd daemon can’t be disabled on the node installation phase (“Day 0”), since this would leave the node without any time reference (and without TLS).
- If PTP sync is lost for a long time, the clock would drift and eventually leave the node not suitable for TLS communications. 

## Solution ##
The configuration files provided here configure `chronyd` to operate dynamically based on the PTP synchronization state. 
The PTP synchronization state is attained from linuxptp-daemon log stored on the node [ptp-sync-check](ptp-sync-check) (this logic will be moved to the `linuxptp-daemon` in the future releases). The sync state is indicated in a marker file stored on the node. This file is used as a condition for starting the `chronyd.service` [20-conditional-start.conf](20-conditional-start.conf). The condition is added to `chronyd.service` as drop-in.
Since the condition is evaluated only when a service starts, there is a `systemd` timer and one-shot service ([chronyd-restart.timer](chronyd-restart.timer) and [chronyd-restart.service](chronyd-restart.service)) periodically restarting the `chronyd.service`.

## Caveats and limitations ##
- Current implementation does not apply any filtering to the raw PTP offset results - the latest offset measurement is used. This can potentially lead to "jumping" in and out of NTP, if the PTP offset has a large jitter around the threshold. This should be fine under the following assumptions:
  - The evaluation  of PTP sync is done once in 5 minutes ([chronyd-restart.timer](chronyd-restart.timer)), which limits the frequency of this potential jumping
  - The threshold is made 5000 times higher ([ptp-sync-check](ptp-sync-check)) than acceptable PTP accuracy for telco workloads. Jumping around this threshold will mean that PTP is unusable, and there is no harm in occasionally jumping to NTP.
- The dropin configuring the conditional start directive implies that the `chronyd` will be periodically restarted.


