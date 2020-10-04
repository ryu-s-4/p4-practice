package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"

	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
)

func main() {
	// コントローラ情報を登録
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
	binary.BigEndian.PutUint16(vlanID, uint16(100))        // TODO: replace with vlan-id what you want.
	macAddr, _ = net.ParseMAC("5e:0b:88:ee:ff:2b")         // TODO: replace with mac addr. what you want.
	binary.BigEndian.PutUint16(portNum, uint16(0))         // TODO: replace with port num. what you want.

	writeRequestInfo.params = append(writeRequestInfo.params, tableid...)
	writeRequestInfo.params = append(writeRequestInfo.params, actionid...)
	writeRequestInfo.params = append(writeRequestInfo.params, vlanID...)
	writeRequestInfo.params = append(writeRequestInfo.params, macAddr...)
	writeRequestInfo.params = append(writeRequestInfo.params, portNum...)

	writeResponse, err = MyWriteRequest(cntlInfo, writeRequestInfo, client)
	if err != nil {
		log.Fatal("write request error.", err)
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
	binary.BigEndian.PutUint16(vlanID, uint16(100))        // TODO: replace with vlan-id what you want.
	macAddr, _ = net.ParseMAC("72:ac:82:22:6e:81")         // TODO: replace with mac addr. what you want.
	binary.BigEndian.PutUint16(portNum, uint16(2))         // TODO: replace with port num. what you want.

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
	binary.BigEndian.PutUint16(vlanID, uint16(100))        // TODO: replace with vlan-id what you want.
	macAddr, _ = net.ParseMAC("ff:ff:ff:ff:ff:ff")         // TODO: replace with mac addr. what you want.
	binary.BigEndian.PutUint16(groupID, uint16(1))         // TODO: replace with group id what you want.

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

	// TODO: マルチキャストグループ登録
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
	writeRequestInfo.params = append(writeRequestInfo.params, groupID...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(0))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	writeRequestInfo.params = append(writeRequestInfo.params, replica...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(1))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	writeRequestInfo.params = append(writeRequestInfo.params, replica...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(2))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	writeRequestInfo.params = append(writeRequestInfo.params, replica...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(3))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	writeRequestInfo.params = append(writeRequestInfo.params, replica...)

	writeResponse, err = MyWriteRequest(cntlInfo, writeRequestInfo, client)
	if err != nil {
		// Error 処理
	}
	log.Printf("WriteResponse: %v", writeResponse)

	// TODO: Read Request で Counter 値を取得
	var readRequestInfo ReadRequestInfo
	counterid := make([]byte, 4)
	index := make([]byte, 8)

	binary.BigEndian.PutUint32(counterid, uint32(99))
	binary.BigEndian.PutUint64(index, uint64(1))
	readRequestInfo.params = append(readRequestInfo.params, counterid...)
	readRequestInfo.params = append(readRequestInfo.params, index...)

	readRequestInfo.entityTypes = make([]string, 0)
	readRequestInfo.entityTypes = append(readRequestInfo.entityTypes, "CounterEntry")
	readRequestInfo.params = make([]byte, 0)
	/*
		counter-ID : byte[0] ~ byte[3]
		index      : byte[4] ~ byte[11]
	*/
	readclient, err := MyReadRequest(cntlInfo, readRequestInfo, client)
	if err != nil {
		// Error 処理
	}

	// Counter 取得（スリープ前）
	var cntentry *v1.CounterEntry
	var readresponse *v1.ReadResponse

	readresponse, err = readclient.Recv()
	if err != nil {
		// Error 処理
	}
	cntentry = readresponse.Entities[0].GetCounterEntry()
	fmt.Println("traffic cnt[in byte]: ", cntentry.Data.ByteCount)

	// スリープ
	fmt.Println("Now Sleeping for 5 seconds.")
	cnt := 1
	for {
		cnt++
		time.Sleep(time.Second * 1)
		if 5 < cnt {
			break
		}
	}
	fmt.Println("Now Getting up.")

	// Counter 取得（スリープ後）
	readclient, err = MyReadRequest(cntlInfo, readRequestInfo, client)
	if err != nil {
		// Error 処理
	}

	readresponse, err = readclient.Recv()
	if err != nil {
		// Error 処理
		log.Fatal("Error. ", err)
	}
	cntentry = readresponse.Entities[0].GetCounterEntry()
	fmt.Println("traffic cnt[in byte]: ", cntentry.Data.ByteCount)
}
