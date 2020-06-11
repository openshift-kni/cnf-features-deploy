# Functional tests


## Discovery mode

To run the tests in discovery mode set the DISCOVERY_MODE environment variable:
```
export DISCOVERY_MODE=true
```

### PTP environment setup for running in discovery mode

The ptp feature needs to be configured to run ptp in discovery mode.
To configure the feature the master and slave PtpConfigs must be created.
Please refer to these exmples for more detail:
- [master PtpConfig example](feature-configs/demo/ptp/ptpconfig-grandmaster.yaml)
- [slave PtpConfig example](feature-configs/demo/ptp/ptpconfig-slave.yaml)

At least 2 nodes must be labeled as grandmaster and slave respectively.
This can be done by labeling the node as follows:
```
oc label node <nodename>  ptp/grandmaster=
oc label node <nodename>  ptp/slave=
```
