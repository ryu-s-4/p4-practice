package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/golang/protobuf/proto"
	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
)

// ControllerInfo is information for the controller
type ControllerInfo struct {
	deviceid    uint64
	roleid      uint64
	electionid  v1.Uint128
	p4infoPath  string
	devconfPath string
}

// WriteRequestInfo is information for WriteRequest.
type WriteRequestInfo struct {
	atomisity  string
	updateType string
	entityType string
	params     []byte
}

// MyMasterArbitrationUpdate gets arbitration for the master
func MyMasterArbitrationUpdate(cntlInfo ControllerInfo, ch v1.P4Runtime_StreamChannelClient) (*v1.MasterArbitrationUpdate, error) {

	update := v1.MasterArbitrationUpdate{
		DeviceId:   cntlInfo.deviceid,
		ElectionId: &cntlInfo.electionid}

	request := v1.StreamMessageRequest{
		Update: &v1.StreamMessageRequest_Arbitration{Arbitration: &update}}

	err := ch.Send(&request)
	if err != nil {
		// Error 処理
		return nil, err
	}

	response, err := ch.Recv()
	if err != nil {
		// Error 処理
		return nil, err
	}

	// response の body は Update 変数（Update()で取得可能）．Update は interface{} 型で下記のいずれか.
	//   - StreamMessageResponse_Arbitration
	//      > Arbitration *MasterArbitrationUpdate
	//   - StreamMessageResponse_Packet
	//      > Packet *PacketIn
	//   - StreamMessageResponse_Digest
	//      > Digest *DigestList
	//   - StreamMessageResponse_IdleTimeoutNotification
	//      > IdleTimeoutNotification *IdleTimeoutNotification
	//   - StreamMessageResponse_Other
	//      > Other *any.Any
	//   - StreamMessageResponse_Error
	//      > Error *StreamError
	// StreamMessageReponse_Arbitration であるか check し，Arbitration を return（GetArbitration()で取得可能）
	updateResponse := response.GetUpdate()
	switch updateResponse.(type) {
	case *v1.StreamMessageResponse_Arbitration:
		arbitrationResponse := response.GetArbitration()
		return arbitrationResponse, nil
	default:
		// Error 処理
		err := fmt.Errorf("Error: Update Type is NOT StreamMessageResponse_Arbitration")
		return nil, err
	}
}

// MyCreateConfig creates config data for SetForwardingPipelineConfig
func MyCreateConfig(p4infoPath string, devconfPath string) (*v1.ForwardingPipelineConfig, error) {

	// create P4Info
	p4info := config_v1.P4Info{}
	p4infoBytes, err := ioutil.ReadFile(p4infoPath)
	if err != nil {
		// Error 処理
	}
	proto.UnmarshalText(string(p4infoBytes), &p4info)

	// create Device Config
	var devconf []byte
	devconf, err = ioutil.ReadFile(devconfPath)
	if err != nil {
		// Error 処理
	}

	// create ForwardingPipelineConfig
	forwardingpipelineconfig := v1.ForwardingPipelineConfig{
		P4Info:         &p4info,
		P4DeviceConfig: devconf}

	return &forwardingpipelineconfig, nil
}

// MySetForwardingPipelineConfig sets the user defined configuration to the data plane.
func MySetForwardingPipelineConfig(cntlInfo ControllerInfo, actionType string, client v1.P4RuntimeClient) (*v1.SetForwardingPipelineConfigResponse, error) {

	var action v1.SetForwardingPipelineConfigRequest_Action
	switch actionType {
	case "VERIFY":
		action = v1.SetForwardingPipelineConfigRequest_VERIFY
	case "VERIFY_AND_SAVE":
		action = v1.SetForwardingPipelineConfigRequest_VERIFY_AND_SAVE
	case "VERIFY_AND_COMMIT":
		action = v1.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT
	case "COMMIT":
		action = v1.SetForwardingPipelineConfigRequest_COMMIT
	case "RECONCILE_AND_COMMIT":
		action = v1.SetForwardingPipelineConfigRequest_RECONCILE_AND_COMMIT
	default:
		action = v1.SetForwardingPipelineConfigRequest_UNSPECIFIED
	}

	config, err := MyCreateConfig(cntlInfo.p4infoPath, cntlInfo.devconfPath)
	if err != nil {
		// Error 処理
	}

	request := v1.SetForwardingPipelineConfigRequest{
		DeviceId:   cntlInfo.deviceid,
		ElectionId: &cntlInfo.electionid,
		Action:     action,
		Config:     config}

	response, err := client.SetForwardingPipelineConfig(context.TODO(), &request)
	if err != nil {
		// Error 処理
	}

	return response, nil
}

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

	for i := 0; (12+8*i) <= len(params); i++ {
		rep = params[(4+8*i):(12+8*i)]
		replica = append(replica, &v1.Replica{ EgressPort: binary.BigEndian.Uint32(rep[0:4]), Instance: binary.BigEndian.Uint32(rep[4:])})
	} 

	multicastGroupEntry := v1.MulticastGroupEntry{
		MulticastGroupId: multicastGroupID,
		Replicas: replica,
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

	case "PacketPacketReplicationEngineEntry":
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
		// Error 処理
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
	updates := make([]*v1.Update, 10)
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
		// Error 処理
	}

	return writeResponse, nil
}

func main() {
	// コントローラ情報を登録
	cntlInfo := ControllerInfo{
		deviceid:    0,
		electionid:  v1.Uint128{High: 0, Low: 1},
		p4infoPath:  "path to p4info file",
		devconfPath: "path to devconf json file ",
	}

	// 接続先サーバーのアドレスとポート番号
	addr := "127.0.0.1"
	port := "20050"

	// gRPC の connection 生成
	conn, err := grpc.Dial(addr+":"+port, grpc.WithInsecure())
	if err != nil {
		log.Fatal("client connection error:", err)
	}
	defer conn.Close()

	// P4runtime Client インスタンス生成
	client := v1.NewP4RuntimeClient(conn)

	// StreamChanel 確立(P4Runtime_StreamChannelClient を return)
	ch, err := client.StreamChannel(context.TODO())
	if err != nil {
		// Error 処理
	}

	// Arbitration 処理（MasterArbitrationUpdate)
	arbitrationResponse, err := MyMasterArbitrationUpdate(cntlInfo, ch)
	if err != nil {
		// Error 処理
	}
	log.Printf("ArbitrationResponse: %v", arbitrationResponse)

	// SetForwardingPipelineConfig 処理
	actionType := "VERIFY_AND_COMMIT"
	setforwardingpipelineconfigResponse, err := MySetForwardingPipelineConfig(cntlInfo, actionType, client)
	if err != nil {
		// Error 処理
	}
	log.Printf("SetForwardingPipelineConfigResponse: %v", setforwardingpipelineconfigResponse)

	// TODO: Write Request でテーブルエントリ登録
	var writeRequestInfo WriteRequestInfo
	var writeResponse *v1.WriteResponse

	tableid := make([]byte, 4)
	actionid := make([]byte, 4)
	vlanID := make([]byte, 2)
	macAddr := make([]byte, 6)
	portNum := make([]byte, 2)
	groupID := make([]byte, 4)
	replica := make([]byte, 8)

	// TODO: MAC テーブル with VLAN にエントリ登録（to host1)
	writeRequestInfo.atomisity = "CONTINUE_ON_ERROR"
	writeRequestInfo.updateType = "INSERT"
	writeRequestInfo.entityType = "TableEntry"
	writeRequestInfo.params = make([]byte, 0)
	/*
		table-ID  : byte[0] ~ byte[3]
		action-ID : byte[4] ~ byte[7]
		VLAN-ID   : byte[8], byte[9]
		MAC       : byte[10] ~ byte[15]
		portNum   : byte[16], byte[17]
	*/

	binary.BigEndian.PutUint32(tableid, uint32(33618152))  // TODO: replace with table id what you want.
	binary.BigEndian.PutUint32(actionid, uint32(16807247)) // TODO: replace with action id what you want.
	binary.BigEndian.PutUint16(vlanID, uint16(100))      // TODO: replace with vlan-id what you want.
	macAddr, _ = net.ParseMAC("c2:ad:c3:95:79:e5")     // TODO: replace with mac addr. what you want.
	binary.BigEndian.PutUint16(portNum, uint16(0))     // TODO: replace with port num. what you want.

	writeRequestInfo.params = append(writeRequestInfo.params, tableid...)
	writeRequestInfo.params = append(writeRequestInfo.params, actionid...)
	writeRequestInfo.params = append(writeRequestInfo.params, vlanID...)
	writeRequestInfo.params = append(writeRequestInfo.params, macAddr...)
	writeRequestInfo.params = append(writeRequestInfo.params, portNum...)

	writeResponse, err = MyWriteRequest(cntlInfo, writeRequestInfo, client)
	if err != nil {
		// Error 処理
	}
	log.Printf("WriteResponse: %v", writeResponse)

	// TODO: MAC テーブル with VLAN にエントリ登録( to host5)
	writeRequestInfo.atomisity = "CONTINUE_ON_ERROR"
	writeRequestInfo.updateType = "INSERT"
	writeRequestInfo.entityType = "TableEntry"
	writeRequestInfo.params = make([]byte, 0)
	/*
		table-ID  : byte[0] ~ byte[3]
		action-ID : byte[4] ~ byte[7]
		VLAN-ID   : byte[8], byte[9]
		MAC       : byte[10] ~ byte[15]
		portNum   : byte[16], byte[17]
	*/

	binary.BigEndian.PutUint32(tableid, uint32(33618152))  // TODO: replace with table id what you want.
	binary.BigEndian.PutUint32(actionid, uint32(16807247)) // TODO: replace with action id what you want.
	binary.BigEndian.PutUint16(vlanID, uint16(100))      // TODO: replace with vlan-id what you want.
	macAddr, _ = net.ParseMAC("aa:92:0a:50:3b:fb")     // TODO: replace with mac addr. what you want.
	binary.BigEndian.PutUint16(portNum, uint16(2))     // TODO: replace with port num. what you want.

	writeRequestInfo.params = append(writeRequestInfo.params, tableid...)
	writeRequestInfo.params = append(writeRequestInfo.params, actionid...)
	writeRequestInfo.params = append(writeRequestInfo.params, vlanID...)
	writeRequestInfo.params = append(writeRequestInfo.params, macAddr...)
	writeRequestInfo.params = append(writeRequestInfo.params, portNum...)

	writeResponse, err = MyWriteRequest(cntlInfo, writeRequestInfo, client)
	if err != nil {
		// Error 処理
	}
	log.Printf("WriteResponse: %v", writeResponse)

	// TODO: ブロードキャストテーブル登録
	writeRequestInfo.atomisity = "CONTINUE_ON_ERROR"
	writeRequestInfo.updateType = "INSERT"
	writeRequestInfo.entityType = "TableEntry"
	writeRequestInfo.params = make([]byte, 0)
	/*
		table-ID  : byte[0] ~ byte[3]
		action-ID : byte[4] ~ byte[7]
		VLAN-ID   : byte[8], byte[9]
		MAC       : byte[10] ~ byte[15]
		group-ID  : byte[16], byte[17]
	*/

	binary.BigEndian.PutUint32(tableid, uint32(33618152))  // TODO: replace with table id what you want.
	binary.BigEndian.PutUint32(actionid, uint32(16791577)) // TODO: replace with action id what you want.
	binary.BigEndian.PutUint16(vlanID, uint16(100))      // TODO: replace with vlan-id what you want.
	macAddr, _ = net.ParseMAC("ff:ff:ff:ff:ff:ff")     // TODO: replace with mac addr. what you want.
	binary.BigEndian.PutUint16(groupID, uint16(1))     // TODO: replace with group id what you want.

	writeRequestInfo.params = append(writeRequestInfo.params, tableid...)
	writeRequestInfo.params = append(writeRequestInfo.params, actionid...)
	writeRequestInfo.params = append(writeRequestInfo.params, vlanID...)
	writeRequestInfo.params = append(writeRequestInfo.params, macAddr...)
	writeRequestInfo.params = append(writeRequestInfo.params, groupID...)

	writeResponse, err = MyWriteRequest(cntlInfo, writeRequestInfo, client)
	if err != nil {
		// Error 処理
	}
	log.Printf("WriteResponse: %v", writeResponse)

	// TODO: Write Request でマルチキャストグループ登録
	writeRequestInfo.atomisity = "CONTINUE_ON_ERROR"
	writeRequestInfo.updateType = "INSERT"
	writeRequestInfo.entityType = "PacketReplicationEngineEntry"
	writeRequestInfo.params = make([]byte, 0)
	/*
		group-ID : byte[0] ~ byte[3]
		replica
		  - egress-port(32bit) : byte[4] ~ byte[7]
		  - instance(32bit)    : byte[8] ~ byte[11]
	*/
	binary.BigEndian.PutUint32(groupID, uint32(1))
	params = append(params, groupID...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(0))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	params = append(params, replica...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(1))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	params = append(params, replica...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(2))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	params = append(params, replica...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(3))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	params = append(params, replica...)

	writeResponse, err = MyWriteRequest(cntlInfo, writeRequestInfo, client)
	if err != nil {
		// Error 処理
	}
	log.Printf("WriteResponse: %v", writeResponse)
	
	// TODO: Write Request で複数の VLAN-ID についてカウンタ値取得，表示
	//   - 無限ループでコマンド受付．show コマンドで一覧表示，など．
}
