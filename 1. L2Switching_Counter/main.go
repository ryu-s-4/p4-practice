package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"

	h "utilities/helper"

	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
)

// SendWriteRequest sends write request to the data plane.
func SendWriteRequest(
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

// CreateReadClient creates New ReadClient.
func CreateReadClient(
	cntlInfo ControllerInfo,
	readreqInfo ReadRequestInfo,
	client v1.P4RuntimeClient) (*v1.P4Runtime_ReadClient, error) {

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

	// コントローラ情報を設定
	cntlInfo := ControllerInfo{
		deviceid:    0,
		electionid:  v1.Uint128{High: 0, Low: 1},
		p4infoPath:  "./switching_p4info.txt",
		devconfPath: "./switching/switching.json",
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

	// P4Info を読み込み
	p4infoText, err := ioutil.ReadFile(cntlinfo.p4infopath)
	if err != nil {
		// Error 処理
	}

	var p4info config_v1.P4Info
	if err := proto.UnmarshalText(string(p4infoText, &p4info); err != nil {
		// Error 処理
	}

	// helper を使って table entry を読み込み
	entries, err := ioutil.ReadFile(runtime_path)
	if err != nil {
		// ReadFile Error
	}

	var entryhelper h.EntryHelper
	if err := json.Unmarshal(entries, &entryhelper); err != nil {
		// Error 処理
	}

	// TableEntry を作成

	// PacketReplicationEngineEntry を作成

	// TableEntry の書き込み

	// sleep １０秒くらい？

	// CounterEntry を作成

	// Counter 値を読み取るための Client を作成し，カウンタ値取得＋表示
}