package event

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"

	jsoniter "github.com/json-iterator/go"

	"github.com/redhat-cne/sdk-go/pkg/event/redfish"
	"github.com/redhat-cne/sdk-go/pkg/types"
)

var iterPool = sync.Pool{
	New: func() interface{} {
		return jsoniter.Parse(jsoniter.ConfigFastest, nil, 1024)
	},
}

func borrowIterator(reader io.Reader) *jsoniter.Iterator {
	iter := iterPool.Get().(*jsoniter.Iterator)
	iter.Reset(reader)
	return iter
}

func returnIterator(iter *jsoniter.Iterator) {
	iter.Error = nil
	iter.Attachment = nil
	iterPool.Put(iter)
}

// ReadJSON ...
func ReadJSON(out *Event, reader io.Reader) error {
	iterator := borrowIterator(reader)
	defer returnIterator(iterator)
	return readJSONFromIterator(out, iterator)
}

// ReadDataJSON ...
func ReadDataJSON(out *Data, reader io.Reader) error {
	iterator := borrowIterator(reader)
	defer returnIterator(iterator)
	return readDataJSONFromIterator(out, iterator)
}

// readDataJSONFromIterator allows you to read the bytes reader as an event
func readDataJSONFromIterator(out *Data, iterator *jsoniter.Iterator) error {
	var (
		// Universally parseable fields.
		version string
		data    []DataValue
		// These fields require knowledge about the specversion to be parsed.
		//schemaurl jsoniter.Any
	)

	for key := iterator.ReadObject(); key != ""; key = iterator.ReadObject() {
		// Check if we have some error in our error cache
		if iterator.Error != nil {
			return iterator.Error
		}

		// If no specversion ...
		switch key {
		case "version":
			version = iterator.ReadString()
		case "values":
			data, _ = readDataValue(iterator)

		default:
			iterator.Skip()
		}
	}

	if iterator.Error != nil {
		return iterator.Error
	}
	out.Version = version
	out.Values = data
	return nil
}

// readJSONFromIterator allows you to read the bytes reader as an event
func readJSONFromIterator(out *Event, iterator *jsoniter.Iterator) error {
	var (
		// Universally parseable fields.
		id     string
		typ    string
		source string
		time   *types.Timestamp
		data   *Data
		err    error

		// These fields require knowledge about the specversion to be parsed.
		//schemaurl jsoniter.Any
	)

	for key := iterator.ReadObject(); key != ""; key = iterator.ReadObject() {
		// Check if we have some error in our error cache
		if iterator.Error != nil {
			return iterator.Error
		}

		// If no specversion ...
		switch key {
		case "id":
			id = iterator.ReadString()
		case "type":
			typ = iterator.ReadString()
		case "source":
			source = iterator.ReadString()
		case "time":
			time = readTimestamp(iterator)
		case "data":
			data, err = readData(iterator)
			if err != nil {
				return err
			}
		case "version":

		case "values":
		//case "DataSchema":
		//schemaurl = iterator.ReadAny()
		default:
			iterator.Skip()
		}
	}

	if iterator.Error != nil {
		return iterator.Error
	}
	out.Time = time
	out.ID = id
	out.Type = typ
	out.Source = source
	if data != nil {
		out.SetData(*data)
	}
	return nil
}

func readTimestamp(iter *jsoniter.Iterator) *types.Timestamp {
	t, err := types.ParseTimestamp(iter.ReadString())
	if err != nil {
		iter.Error = err
	}
	return t
}

func readDataValue(iter *jsoniter.Iterator) ([]DataValue, error) {
	var values []DataValue
	var err error
	for iter.ReadArray() {
		var cacheValue interface{}
		dv := DataValue{}
		for dvField := iter.ReadObject(); dvField != ""; dvField = iter.ReadObject() {
			switch dvField {
			case "resource":
				dv.Resource = iter.ReadString()
			case "dataType":
				dv.DataType = DataType(iter.ReadString())
			case "valueType":
				dv.ValueType = ValueType(iter.ReadString())
			case "value":
				cacheValue = iter.Read()
			default:
				iter.Skip()
			}
		}

		if dv.ValueType == DECIMAL {
			dv.Value, err = strconv.ParseFloat(cacheValue.(string), 64)
		} else if dv.ValueType == ENUMERATION {
			dv.Value = cacheValue.(string)
		} else if dv.ValueType == REDFISH_EVENT {
			jsonRedfishEvent, err2 := json.Marshal(cacheValue)
			if err2 != nil {
				return values, err2
			}

			e := redfish.Event{}
			if err = json.Unmarshal(jsonRedfishEvent, &e); err != nil {
				return values, err
			}
			dv.Value = e
		} else {
			return values, fmt.Errorf("value type %v is not supported", dv.ValueType)
		}
		values = append(values, dv)
	}

	return values, err
}

func readData(iter *jsoniter.Iterator) (*Data, error) {
	data := &Data{
		Version: "",
		Values:  []DataValue{},
	}

	for key := iter.ReadObject(); key != ""; key = iter.ReadObject() {
		// Check if we have some error in our error cache
		if iter.Error != nil {
			return data, iter.Error
		}
		switch key {
		case "version":
			data.Version = iter.ReadString()
		case "values":
			values, err := readDataValue(iter)
			if err != nil {
				return data, err
			}
			data.Values = values
		default:
			iter.Skip()
		}
	}

	return data, nil
}

// UnmarshalJSON implements the json unmarshal method used when this type is
// unmarshaled using json.Unmarshal.
func (e *Event) UnmarshalJSON(b []byte) error {
	iterator := jsoniter.ConfigFastest.BorrowIterator(b)
	defer jsoniter.ConfigFastest.ReturnIterator(iterator)
	return readJSONFromIterator(e, iterator)
}

// UnmarshalJSON implements the json unmarshal method used when this type is
// unmarshaled using json.Unmarshal.
func (d *Data) UnmarshalJSON(b []byte) error {
	iterator := jsoniter.ConfigFastest.BorrowIterator(b)
	defer jsoniter.ConfigFastest.ReturnIterator(iterator)
	return readDataJSONFromIterator(d, iterator)
}
