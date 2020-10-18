package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"./myutils"

	"github.com/golang/protobuf/proto"
	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

// MyMasterArbitrationUpdate gets arbitration for the master
func MyMasterArbitrationUpdate(cntlInfo ControllerInfo, ch v1.P4Runtime_StreamChannelClient) (*v1.MasterArbitrationUpdate, error) {

	request := v1.StreamMessageRequest{
		Update: &v1.StreamMessageRequest_Arbitration{
			Arbitration: &v1.MasterArbitrationUpdate{
				DeviceId:   cntlInfo.deviceid,
				ElectionId: &cntlInfo.electionid,
			},
		},
	}

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
		arbitration := response.GetArbitration()
		return arbitration, nil
	/*
		case *v1.StreamMessageResponse_Packet:
			packet := response.GetPacket()
		case *v1.StreamMessageResponse_Digest:
			digest := response.GetDigest()
		case *v1.StreamMessageResponse_IdleTimeoutNotification:
			idletimenotf := response.GetIdleTimeoutNotification()
	*/
	default:
		err := fmt.Errorf("Error: not supported. ")
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

// SendWriteRequest sends write request to the data plane.
func SendWriteRequest(
	cntlInfo ControllerInfo,
	updates []*v1.Update,
	atomisityType string,
	client v1.P4RuntimeClient) (*v1.WriteResponse, error) {

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
		log.Fatal("Error at MyWriteRequest. ", err)
	}

	return writeResponse, nil
}

// CreateReadClient creates New ReadClient.
func CreateReadClient(
	cntlInfo ControllerInfo,
	entities []*v1.Entity,
	client v1.P4RuntimeClient) (*v1.P4Runtime_ReadClient, error) {

	readRequest := v1.ReadRequest{
		DeviceId: cntlInfo.deviceid,
		Entities: entities,
	}

	readclient, err := client.Read(context.TODO(), &readRequest)
	if err != nil {
		// Error 処理
	}

	return &readclient, nil
}

// ControllerInfo is information for the controller
type ControllerInfo struct {
	deviceid    uint64
	roleid      uint64
	electionid  v1.Uint128
	p4infoPath  string
	devconfPath string
}

func main() {

	/*

		// コントローラ情報を設定
		cntlInfo := ControllerInfo{
			deviceid:    0,
			electionid:  v1.Uint128{High: 0, Low: 1},
			p4infoPath:  "./p4info.txt",
			devconfPath: "./switching.json",
		}

		// 接続先サーバーのアドレスとポート番号
		addr := "127.0.0.1"
		port := "50051"

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

	*/

	// P4Info を読み込み
	p4infoText, err := ioutil.ReadFile(cntlInfo.p4infoPath)
	if err != nil {
		// Error 処理
	}
	fmt.Println("INFO: ReadFile successfully.") // FOR DEBUG

	var p4info config_v1.P4Info
	if err := proto.UnmarshalText(string(p4infoText), &p4info); err != nil {
		// Error 処理
		log.Fatal("Error: cannot unmarshal p4info.txt.", err)
	}
	fmt.Println("INFO: Unmarshal P4Info.txt successfully.") // FOR DUBUG

	// helper を使って table entry を読み込み
	entries, err := ioutil.ReadFile(cntlInfo.devconfPath)
	if err != nil {
		// ReadFile Error
		log.Fatal("Error: cannot read file (runtime).")
	}

	var entryhelper helper.EntryHelper
	if err := json.Unmarshal(entries, &entryhelper); err != nil {
		// Error 処理
	}
	fmt.Println("INFO: Unmarshal runtime file successfully.") // FOR DEBUG

	// Update 定義
	var updates []*v1.Update

	// Entity を作成
	for tableentryhelper := range entryhelper.TableEntries {
		tableentry, err := helper.BuildTableEntry(tableentryhelper, p4info)
		if err != nil {
			// Error 処理
		}
		update, err := helper.NewUpdate("INSERT", tableentry)
		if err != nil {
			// Error 処理
		}
		updates = append(updates, update)
	}

	for multicastgroupentryhelper := range entryhelper.MulticastGroupEntries {
		multicastgroupentry, err := helper.BuildMulticastGroupEntry(multicastgroupentryhelper, p4info)
		if err != nil {
			// Error 処理
		}
		update, err := helper.NewUpdate("INSERT", multicastgroupentry)
		if err != nil {
			// Error 処理
		}
		updates = append(updates, update)
	}

	/*

		// Entity の書き込み
		writeResponse, err := SendWriteRequest(cntlInfo, updates, "CONTINUE_ON_ERROR", client)
		if err != nil {
			// Error 処理
		}
		log.Printf("WriteResponse: %v", writeResponse)

		// sleep １０秒くらい？

		// CounterEntry を作成

		// Counter 値を読み取るための Client を作成し，カウンタ値取得＋表示

	*/

}
