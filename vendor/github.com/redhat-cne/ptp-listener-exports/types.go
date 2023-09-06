package exports

import (
	"sync"
)

const Port9085 = 9085

type StoredEvent map[string]interface{}

const (
	EventTimeStamp = "eventtimestamp"
	EventType      = "eventtype"
	EventSource    = "eventsource"
	EventValues    = "eventvalues"
)

type StoredEventValues map[string]interface{}

var (
	Mu        sync.Mutex
	AllEvents []StoredEvent
)
