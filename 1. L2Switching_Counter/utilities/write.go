package write

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"

	v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

// MyNewTableEntry creates new TableEntry instance
func MyNewTableEntry(params []byte) *v1.Entity_TableEntry {

	tableID := binary.BigEndian.Uint32(params[0:4])
	actionID := binary.BigEndian.Uint32(params[4:8])

	vlanID := params[8:10]
	macAddr := params[10:16]
	portNum_or_groupID := params[16:18]

	tableEntry := v1.TableEntry{
		TableId: tableID,
		Match: []*v1.FieldMatch{
			{
				FieldId: uint32(1),
				FieldMatchType: &v1.FieldMatch_Exact_{
					Exact: &v1.FieldMatch_Exact{
						Value: vlanID,
					},
				},
			},
			{
				FieldId: uint32(2),
				FieldMatchType: &v1.FieldMatch_Exact_{
					Exact: &v1.FieldMatch_Exact{
						Value: macAddr,
					},
				},
			},
		},
		Action: &v1.TableAction{
			Type: &v1.TableAction_Action{
				Action: &v1.Action{
					ActionId: actionID,
					Params: []*v1.Action_Param{
						{
							ParamId: uint32(1),
							Value:   portNum_or_groupID,
						},
					},
				},
			},
		},
	}

	entityTableEntry := v1.Entity_TableEntry{
		TableEntry: &tableEntry}

	return &entityTableEntry
}

// MyNewMulticastGroupEntry creates new MulticastGroupEntry instanse.
func MyNewMulticastGroupEntry(params []byte) *v1.MulticastGroupEntry {

	multicastGroupID := binary.BigEndian.Uint32(params[0:4])

	var replica []*v1.Replica
	rep := make([]byte, 8)

	for i := 0; (12 + 8*i) <= len(params); i++ {
		rep = params[(4 + 8*i):(12 + 8*i)]
		replica = append(replica, &v1.Replica{EgressPort: binary.BigEndian.Uint32(rep[0:4]), Instance: binary.BigEndian.Uint32(rep[4:])})
	}

	multicastGroupEntry := v1.MulticastGroupEntry{
		MulticastGroupId: multicastGroupID,
		Replicas:         replica,
	}

	return &multicastGroupEntry
}

// MyNewPacketReplicationEngineEntry creates new PacketReplicationEngineEntry instanse.
func MyNewPacketReplicationEngineEntry(params []byte) *v1.Entity_PacketReplicationEngineEntry {

	multicastGroupEntry := MyNewMulticastGroupEntry(params)

	packetReplicationEngineEntry_MulticastGroupEntry := v1.PacketReplicationEngineEntry_MulticastGroupEntry{
		MulticastGroupEntry: multicastGroupEntry,
	}

	packetReplicationEngineEntry := v1.PacketReplicationEngineEntry{
		Type: &packetReplicationEngineEntry_MulticastGroupEntry,
	}

	entity_PacketReplicationEngineEntry := v1.Entity_PacketReplicationEngineEntry{
		PacketReplicationEngineEntry: &packetReplicationEngineEntry,
	}

	return &entity_PacketReplicationEngineEntry
}

// MyNewEntry creates new Entry instance
func MyNewEntry(entityType string, params []byte) (*v1.Entity, error) {

	// return Entry based on entityType.
	switch entityType {
	case "TableEntry":
		ent := MyNewTableEntry(params)
		entity := v1.Entity{Entity: ent}
		return &entity, nil

	case "PacketReplicationEngineEntry":
		ent := MyNewPacketReplicationEngineEntry(params)
		entity := v1.Entity{Entity: ent}
		return &entity, nil

	default:
		err := fmt.Errorf("Error: %s is NOT supported", entityType)
		return nil, err
	}
}

// MyNewUpdate creates new Update instance.
func MyNewUpdate(updateType string, entityType string, params []byte) (*v1.Update, error) {

	entity, err := MyNewEntry(entityType, params)
	if err != nil {
		log.Fatal("Error at MyNewUpdate.", err)
		return nil, err
	}

	switch updateType {
	case "INSERT":
		update := v1.Update{
			Type:   v1.Update_INSERT,
			Entity: entity}
		return &update, nil

	case "MODIFY":
		update := v1.Update{
			Type:   v1.Update_MODIFY,
			Entity: entity}
		return &update, nil

	case "DELETE":
		update := v1.Update{
			Type:   v1.Update_DELETE,
			Entity: entity}
		return &update, nil

	default:
		update := v1.Update{
			Type:   v1.Update_UNSPECIFIED,
			Entity: entity}
		return &update, nil
	}
}

// MyWriteRequest sends write request to the data plane.
func MyWriteRequest(
	cntlInfo ControllerInfo,
	writeRequestInfo WriteRequestInfo,
	client v1.P4RuntimeClient) (*v1.WriteResponse, error) {

	update, err := MyNewUpdate(writeRequestInfo.updateType, writeRequestInfo.entityType, writeRequestInfo.params)
	updates := make([]*v1.Update, 0)
	updates = append(updates, update)

	var atomisity v1.WriteRequest_Atomicity
	switch writeRequestInfo.atomisity {
	case "CONTINUE_ON_ERROR":
		atomisity = v1.WriteRequest_CONTINUE_ON_ERROR
	case "ROLLBACK_ON_ERROR": // Optional
		atomisity = v1.WriteRequest_ROLLBACK_ON_ERROR
	case "DATAPLANE_ATOMIC": // Optional
		atomisity = v1.WriteRequest_DATAPLANE_ATOMIC
	default:
		atomisity = v1.WriteRequest_CONTINUE_ON_ERROR
	}

	writeRequest := v1.WriteRequest{
		DeviceId:   cntlInfo.deviceid,
		ElectionId: &cntlInfo.electionid,
		Updates:    updates,
		Atomicity:  atomisity,
	}

	writeResponse, err := client.Write(context.TODO(), &writeRequest)
	if err != nil {
		log.Fatal("Error at MyWriteRequest. ", err)
	}

	return writeResponse, nil
}
