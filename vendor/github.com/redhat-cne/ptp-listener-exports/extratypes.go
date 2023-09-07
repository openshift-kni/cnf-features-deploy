package exports

import (
	ptpEvent "github.com/redhat-cne/sdk-go/pkg/event/ptp"
)

// Conversion for LockState event enumeration to int
type LockStateValue int64

const (
	AcquiringSync LockStateValue = iota
	AntennaDisconnected
	AntennaShortCircuit
	Booting
	Freerun
	Holdover
	Locked
	Synchronized
	Unlocked
)

var (
	ToLockStateValue = map[string]LockStateValue{
		string(ptpEvent.ACQUIRING_SYNC):        AcquiringSync,
		string(ptpEvent.ANTENNA_DISCONNECTED):  AntennaDisconnected,
		string(ptpEvent.ANTENNA_SHORT_CIRCUIT): AntennaShortCircuit,
		string(ptpEvent.BOOTING):               Booting,
		string(ptpEvent.FREERUN):               Freerun,
		string(ptpEvent.HOLDOVER):              Holdover,
		string(ptpEvent.LOCKED):                Locked,
		string(ptpEvent.SYNCHRONIZED):          Synchronized,
		string(ptpEvent.UNLOCKED):              Unlocked,
	}
)
