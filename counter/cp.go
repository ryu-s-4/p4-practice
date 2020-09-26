package main

import (
	"context"
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
func MyNewTableEntry(params []byte) (v1.Entity_TableEntry, error) {

	tableID := uint32(99)                             // TODO: replace with table id from p4info file.
	vlanID := []byte{uint8(120)}                      // TODO: replace with vlan-id what you want.
	macAddr, err := net.ParseMAC("00:11:22:33:44:55") // TODO: replace with mac addr. what you want.
	if err != nil {
		// Error 処理
	}
	actionID := uint32(99)                // TODO: replace with action id from p4info file.
	portNum := []byte{uint8(0), uint8(1)} // TODO: replace with port num. what you want.
	// groupID := []byte{uint8(0), uint8(1)} // TODO: replace with group id what you want.

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
							Value:   portNum, // or groupID
						},
					},
				},
			},
		},
	}

	entityTableEntry := v1.Entity_TableEntry{
		TableEntry: &tableEntry}

	return entityTableEntry, nil
}

// MyNewEntry creates new Entry instance
func MyNewEntry(entityType string, params []byte) (*v1.Entity, error) {

	// return Entry based on entityType.
	switch entityType {
	case "TableEntry":
		ent, err := MyNewTableEntry(params)
		if err != nil {
			// Error 処理
		}
		entity := v1.Entity{Entity: &ent}
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
	atomisityType string,
	updateType string,
	entityType string,
	params []byte,
	client v1.P4RuntimeClient) (*v1.WriteResponse, error) {

	update, err := MyNewUpdate(updateType, entityType, params)
	updates := make([]*v1.Update, 10)
	updates = append(updates, update)

	var atomisity v1.WriteRequest_Atomicity
	switch atomisityType {
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

	// TODO: Write Request で MAC テーブルにエントリ登録
	atomisity := "CONTINUE_ON_ERROR"
	updateType := "INSERT"
	entityType := "TableEntry"
	params := make([]byte, 10)

	writeResponse, err := MyWriteRequest(cntlInfo, atomisity, updateType, entityType, params, client)
	if err != nil {
		// Error 処理
	}
	log.Printf("WriteResponse: %v", writeResponse)

	// TODO: Write Request でブロードキャストテーブル登録（マルチキャストグループIDとVLANID+ブロードキャストアドレスの紐つけ）

	// TODO: Write Request でマルチキャストグループ登録
	//   - v1.MulticastGroupEntry の Writerequest??
	//   - multicast_group_id と replica を設定する？

	// TODO: Write Request で複数の VLAN-ID についてカウンタ値取得，表示
	//   - 無限ループでコマンド受付．show コマンドで一覧表示，など．
}
