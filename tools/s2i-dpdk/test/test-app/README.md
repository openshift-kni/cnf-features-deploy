# DPDK workload test app

This folder contains a simple dpdk workload test.
The test application use `expect` to run `testpmd` and then sleeps forever.

## Expected outputs

The build log shows how the app is built:

```bash
$ oc logs build/s2i-dpdk-1

Caching blobs under "/var/cache/blobs".
Getting image source signatures
Copying blob sha256:<output omitted>
Writing manifest to image destination
Storing signatures
Generating dockerfile with builder image quay.io/openshift-kni/dpdk:4.5
STEP 1: FROM quay.io/openshift-kni/dpdk:4.5
STEP 2: LABEL <output omitted>
STEP 3: ENV <output omitted>
STEP 4: USER root
STEP 5: COPY upload/src /tmp/src
STEP 6: RUN chown -R 1001:0 /tmp/src
STEP 7: USER 1001
STEP 8: RUN /usr/libexec/s2i/assemble
---> Installing application source...
---> Building application from source...
make: Entering directory `/opt/app-root/src/test-pmd'
  CC testpmd.o
<output omitted>
  LD testpmd
  INSTALL-APP testpmd
  INSTALL-MAP testpmd.map
make: Leaving directory `/opt/app-root/src/test-pmd'
build done
STEP 9: CMD /usr/libexec/s2i/run
STEP 10: COMMIT temp.builder.openshift.io/dpdk/s2i-dpdk-1:2e14641b
<output omitted>
Successfully pushed image-registry.openshift-image-registry.svc:5000/dpdk/s2i-dpdk-app@sha256:3d6e8628577250a49d4ff8336ea277062b5f900940698833ea5b2de68d6d3788
Push successful
```

The demo app expected output is similar to:

```
$ oc logs s2i-dpdk-app-1-ksz5w
++ cat /sys/fs/cgroup/cpuset/cpuset.cpus
+ export CPU=0,2,28,30
+ CPU=0,2,28,30
+ echo 0,2,28,30
0,2,28,30
+ echo 0000:01:10.0
0000:01:10.0
+ '[' testpmd == testpmd ']'
+ envsubst
+ chmod +x test.sh
+ expect -f test.sh
spawn ./customtestpmd -l 0,2,28,30 -w 0000:01:10.0 --iova-mode=va -- -i --portmask=0x1 --nb-cores=2 --forward-mode=mac --port-topology=loop --no-mlockall
EAL: Detected 56 lcore(s)
EAL: Detected 2 NUMA nodes
EAL: Multi-process socket /var/run/dpdk/rte/mp_socket
EAL: Selected IOVA mode 'VA'
EAL: No free hugepages reported in hugepages-1048576kB
EAL: Probing VFIO support...
EAL: VFIO support initialized
EAL: PCI device 0000:01:10.0 on NUMA socket 0
EAL:   probe driver: 8086:1515 net_ixgbe_vf
EAL:   using IOMMU type 1 (Type 1)
Interactive-mode selected
Set mac packet forwarding mode
testpmd: create a new mbuf pool <mbuf_pool_socket_0>: n=171456, size=2176, socket=0
testpmd: preferred mempool ops selected: ring_mp_mc
Configuring Port 0 (socket 0)
Port 0: 02:09:C0:D2:0F:6A
Checking link statuses...
Done
testpmd> start
mac packet forwarding - ports=1 - cores=1 - streams=1 - NUMA support enabled, MP allocation mode: native
Logical Core 2 (socket 0) forwards packets on 1 streams:
  RX P=0/Q=0 (socket 0) -> TX P=0/Q=0 (socket 0) peer=02:00:00:00:00:00

  mac packet forwarding packets/burst=32
  nb forwarding cores=2 - nb forwarding ports=1
  port 0: RX queue number: 1 Tx queue number: 1
    Rx offloads=0x0 Tx offloads=0x0
    RX queue: 0
      RX desc=512 - RX free threshold=32
      RX threshold registers: pthresh=8 hthresh=8  wthresh=0
      RX Offloads=0x0
    TX queue: 0
      TX desc=512 - TX free threshold=32
      TX threshold registers: pthresh=32 hthresh=0  wthresh=0
      TX offloads=0x0 - TX RS bit threshold=32
testpmd> stop
Telling cores to stop...
Waiting for lcores to finish...

  ---------------------- Forward statistics for port 0  ----------------------
  RX-packets: 0              RX-dropped: 0             RX-total: 0
  TX-packets: 0              TX-dropped: 0             TX-total: 0
  ----------------------------------------------------------------------------

  +++++++++++++++ Accumulated forward statistics for all ports+++++++++++++++
  RX-packets: 0              RX-dropped: 0             RX-total: 0
  TX-packets: 0              TX-dropped: 0             TX-total: 0
  ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

Done.
testpmd> quit

Stopping port 0...
Stopping ports...
Done

Shutting down port 0...
Closing ports...
Done

Bye...
+ true
+ sleep inf
```
