spawn ./customtestpmd -l ${CPU} -w ${PCIDEVICE_OPENSHIFT_IO_DPDKNIC} --iova-mode=va -- -i --portmask=0x1 --nb-cores=2 --forward-mode=mac --port-topology=loop --no-mlockall
set timeout 10000
expect "testpmd>"
send -- "port stop 0\r"
expect "testpmd>"
send -- "port detach 0\r"
expect "testpmd>"
send -- "port attach ${PCIDEVICE_OPENSHIFT_IO_DPDKNIC}\r"
expect "testpmd>"
send -- "port start 0\r"
expect "testpmd>"
send -- "start\r"
expect "testpmd>"
sleep 30
send -- "stop\r"
expect "testpmd>"
send -- "quit\r"
expect eof
