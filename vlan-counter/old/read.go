package read

import (
	"context"

	v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

// MyNewCounterEntry creates new CounterEntry instance.
func MyNewCounterEntry(params []byte) *v1.CounterEntry {

	counterEntry := v1.CounterEntry{
		CounterId: uint32(302003629),
		Index: &v1.Index{
			Index: int64(100),
		},
	}
	return &counterEntry
}

// MyReadRequest get read client for getting the data from data plane.
func MyReadRequest(
	cntlInfo ControllerInfo,
	readreqInfo ReadRequestInfo,
	client v1.P4RuntimeClient) (v1.P4Runtime_ReadClient, error) {

	var entity v1.Entity
	entities := make([]*v1.Entity, 0)

	entity = v1.Entity{
		Entity: &v1.Entity_CounterEntry{
			CounterEntry: MyNewCounterEntry(readreqInfo.params),
		},
	}
	entities = append(entities, &entity)

	readRequest := v1.ReadRequest{
		DeviceId: cntlInfo.deviceid,
		Entities: entities,
	}

	readclient, err := client.Read(context.TODO(), &readRequest)
	if err != nil {
		// Error 処理
	}

	return readclient, nil
}
