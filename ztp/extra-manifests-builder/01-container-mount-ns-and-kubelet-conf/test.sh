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

# Mock for 'systemctl`; mimics the actual output as best we can.
systemctl() {
    local action=$1 arg=$2
    local file=$TMPDIR/$arg
    if [[ ! -f $file ]]; then
        echo "No files found for $arg." >&2
        return 1
    fi
    echo "# $file"
    cat $file
}
export -f systemctl

cat >$TMPDIR/dbus.service <<EOF
[Unit]
Description=D-Bus System Message Bus
Documentation=man:dbus-broker-launch(1)
DefaultDependencies=false
After=dbus.socket
Before=basic.target shutdown.target
Requires=dbus.socket
Conflicts=shutdown.target

[Service]
Type=notify
Sockets=dbus.socket
OOMScoreAdjust=-900
LimitNOFILE=16384
ProtectSystem=full
PrivateTmp=true
PrivateDevices=true
ExecStart=/usr/bin/dbus-broker-launch --scope system --audit
ExecReload=/usr/bin/busctl call org.freedesktop.DBus /org/freedesktop/DBus org.freedesktop.DBus ReloadConfig

[Install]
Alias=dbus.service
EOF

echo "Test: Missing service"
result=$(./extractExecStart nosuchfile.service)
[[ $? -eq 0 ]] || fatal "extractExecStart unexpected exit code $?"
[[ $result == "" ]] || fatal "Expected extractExecStart on a missing service to fail"

echo "Test: Stdout return"
result=$(./extractExecStart dbus.service)
[[ $? -eq 0 ]] || fatal "extractExecStart unexpected exit code $?"
[[ $result == "/usr/bin/dbus-broker-launch --scope system --audit" ]] || fatal "Unexpected result: $result"

echo "Test: Write to file"
./extractExecStart dbus.service $TMPDIR/one.env
[[ $? -eq 0 ]] || fatal "extractExecStart unexpected exit code $?"
result=$(<$TMPDIR/one.env)
[[ $result == "EXECSTART=/usr/bin/dbus-broker-launch --scope system --audit" ]] || fatal "Unexpected result: $result"

echo "Test: Write to file with custom env variable name"
./extractExecStart dbus.service $TMPDIR/one.env FOOBAR
[[ $? -eq 0 ]] || fatal "extractExecStart unexpected exit code $?"
result=$(<$TMPDIR/one.env)
[[ $result == "FOOBAR=/usr/bin/dbus-broker-launch --scope system --audit" ]] || fatal "Unexpected result: $result"

echo "All test completed successfully"
