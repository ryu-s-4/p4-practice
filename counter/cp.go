package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/golang/protobuf/proto"
	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
)

// MyMasterArbitrationUpdate gets arbitration for the master
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

// MyCreateConfig creates config data for SetForwardingPipelineConfig
func MyCreateConfig(p4infoPath string, devconfPath string) (v1.ForwardingPipelineConfig, error) {

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

	return forwardingpipelineconfig, nil

}

func main() {
	// コントローラ情報を登録
	type ControllerInfo struct {
		deviceid   uint64
		roleid     uint64
		electionid v1.Uint128
	}

	cntlInfo := ControllerInfo{
		deviceid:   0,
		electionid: v1.Uint128{High: 0, Low: 1}}

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
	p4infoPath := "pathtop4info"
	devconfPath := "pathtodevconf"

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
	config, err := MyCreateConfig(p4infoPath, devconfPath)
	if err != nil {
		// Error 処理
	}

	request := v1.SetForwardingPipelineConfigRequest{
		DeviceId:   cntlInfo.deviceid,
		ElectionId: &cntlInfo.electionid,
		Action:     action,
		Config:     &config}

	response, err := client.SetForwardingPipelineConfig(context.TODO(), &request)
	if err != nil {
		// Error 処理
	} else {
		log.Printf("SetForwardingPipelineConfig_Response Info: %v", *response)
	}

	// TODO: Write Request で MAC テーブルにエントリ登録

	// TODO: Write Request でマルチキャストグループ登録
	/*
		v1.MulticastGroupEntry の Writerequest??
		multicast_group_id と replica を設定する？
	*/

	// TODO: Write Request でブロードキャストテーブル登録（マルチキャストグループIDとVLANID+ブロードキャストアドレスの紐つけ）

	// TODO: Write Request で複数の VLAN-ID についてカウンタ値取得，表示
	/*
		無限ループでコマンド受付．show コマンドで一覧表示，など．
	*/
}
