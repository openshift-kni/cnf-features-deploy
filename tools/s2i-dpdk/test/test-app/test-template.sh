spawn testpmd -l ${CPU} -w ${PCIDEVICE_OPENSHIFT_IO_DPDKNIC}  -- -i --portmask=0x1 --nb-cores=2 --forward-mode=mac --port-topology=loop
set timeout 10000
expect "testpmd>"
send -- "start\r"
sleep 20
expect "testpmd>"
send -- "stop\r"
expect "testpmd>"
send -- "quit\r"
expect eof
