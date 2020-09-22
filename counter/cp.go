package main

import (
	"context"
	"fmt"
	"log"

	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
)

// MyMasterArbitrationUpdate : get arbitration for the master
func MyMasterArbitrationUpdate(ch v1.P4Runtime_StreamChannelClient, update *v1.MasterArbitrationUpdate) (*v1.MasterArbitrationUpdate, error) {
	request := v1.StreamMessageRequest{
		Update: &v1.StreamMessageRequest_Arbitration{Arbitration: update}}

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
		ArbitrationResponse := response.GetArbitration()
		return ArbitrationResponse, nil
	default:
		// Error 処理
		err := fmt.Errorf("Error: Update Type is NOT StreamMessageResponse_Arbitration")
		return nil, err
	}
}

func main() {
	// コントローラ（クライアント）を作成
	type ControllerInfo struct {
		deviceid   uint64
		roleid     uint64
		electionid v1.Uint128
	}

	cntlInfo := ControllerInfo{
		deviceid:   0,
		electionid: v1.Uint128{High: 0, Low: 1}}

	// P4Info 作成


	
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
	update := v1.MasterArbitrationUpdate{
		DeviceId:   cntlInfo.deviceid,
		ElectionId: &cntlInfo.electionid}

	arbitration, err := MyMasterArbitrationUpdate(ch, &update)
	if err != nil {
		// Error 処理
	} else {
		log.Printf("Arbitration Info: %v", *arbitration)
	}

	// SetForwardingPipelineConfig 処理
	action := v1.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT
	/*  
	const (
		SetForwardingPipelineConfigRequest_UNSPECIFIED SetForwardingPipelineConfigRequest_Action = 0
		SetForwardingPipelineConfigRequest_VERIFY SetForwardingPipelineConfigRequest_Action = 1
		SetForwardingPipelineConfigRequest_VERIFY_AND_SAVE SetForwardingPipelineConfigRequest_Action = 2
		SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT SetForwardingPipelineConfigRequest_Action = 3
		SetForwardingPipelineConfigRequest_COMMIT SetForwardingPipelineConfigRequest_Action = 4
		SetForwardingPipelineConfigRequest_RECONCILE_AND_COMMIT SetForwardingPipelineConfigRequest_Action = 5
	)
	*/ 
	config := // input config file
	request := v1.SetForwardingPipelineConfigRequest{
		DeviceId: cntlInfo.deviceid, 
		ElectionId: cntlInfo.electionid, 
		Action: action, 
		Config: config}
	// Write Request で複数の VLAN-ID についてカウンタ値取得

	// カウンタ値表示
}
