package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/p4-practice/vlan-counter/myutils"

	"github.com/golang/protobuf/proto"
	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
)

// MasterArbitrationUpdate gets arbitration for the master
func MasterArbitrationUpdate(cntlInfo ControllerInfo, ch v1.P4Runtime_StreamChannelClient) (*v1.MasterArbitrationUpdate, error) {

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
		return nil, err
	}

	response, err := ch.Recv()
	if err != nil {
		return nil, err
	}

	updateResponse := response.GetUpdate()
	switch updateResponse.(type) {

	case *v1.StreamMessageResponse_Arbitration:
		arbitration := response.GetArbitration()
		return arbitration, nil

	case *v1.StreamMessageResponse_Packet:
		/* TODO */
		/* packet := response.GetPacket() */

	case *v1.StreamMessageResponse_Digest:
		/* TODO */
		/* digest := response.GetDigest() */

	case *v1.StreamMessageResponse_IdleTimeoutNotification:
		/* TODO */
		/* idletimenotf := response.GetIdleTimeoutNotification() */
	}

	/* unknown update response type is received. */
	return nil, fmt.Errorf("unknown update response type")
}

// MyCreateConfig creates config data for SetForwardingPipelineConfig
func MyCreateConfig(p4infoPath string, devconfPath string) (*v1.ForwardingPipelineConfig, error) {

	p4info := config_v1.P4Info{}
	p4infoBytes, err := ioutil.ReadFile(p4infoPath)
	if err != nil {
		return nil, err
	}
	proto.UnmarshalText(string(p4infoBytes), &p4info)

	var devconf []byte
	devconf, err = ioutil.ReadFile(devconfPath)
	if err != nil {
		return nil, err
	}

	forwardingpipelineconfig := v1.ForwardingPipelineConfig{
		P4Info:         &p4info,
		P4DeviceConfig: devconf}

	return &forwardingpipelineconfig, nil
}

// SetForwardingPipelineConfig sets the user defined configuration to the data plane.
func SetForwardingPipelineConfig(
	cntlInfo ControllerInfo,
	actionType string,
	client v1.P4RuntimeClient) (*v1.SetForwardingPipelineConfigResponse, error) {

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
		return nil, err
	}

	request := v1.SetForwardingPipelineConfigRequest{
		DeviceId:   cntlInfo.deviceid,
		ElectionId: &cntlInfo.electionid,
		Action:     action,
		Config:     config}

	response, err := client.SetForwardingPipelineConfig(context.TODO(), &request)
	if err != nil {
		return nil, err
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
	case "ROLLBACK_ON_ERROR": // OPTIONAL
		atomisity = v1.WriteRequest_ROLLBACK_ON_ERROR
	case "DATAPLANE_ATOMIC": // OPTIONAL
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
		return nil, err
	}

	return writeResponse, nil
}

// CreateReadClient creates New ReadClient.
func CreateReadClient(
	cntlInfo ControllerInfo,
	entities []*v1.Entity,
	client v1.P4RuntimeClient) (v1.P4Runtime_ReadClient, error) {

	readRequest := v1.ReadRequest{
		DeviceId: cntlInfo.deviceid,
		Entities: entities,
	}

	readclient, err := client.Read(context.TODO(), &readRequest)
	if err != nil {
		return nil, err
	}

	return readclient, nil
}

// ControllerInfo is information for the controller
type ControllerInfo struct {
	deviceid    uint64
	roleid      uint64
	electionid  v1.Uint128
	p4infoPath  string
	devconfPath string
	runconfPath string
}

func main() {

	/* 各種情報を設定 */
	cntlInfo := ControllerInfo{
		deviceid:    0,
		electionid:  v1.Uint128{High: 0, Low: 1},
		p4infoPath:  "./p4info.txt",
		devconfPath: "./switching.json",
		runconfPath: "./runtime.json",
	}
	addr := "127.0.0.1" /* gRPC サーバのアドレス */
	port := "50051"     /* gRPC サーバのポート番号 */

	/* gRPC connection 確立 */
	conn, err := grpc.Dial(addr+":"+port, grpc.WithInsecure())
	if err != nil {
		log.Fatal("ERROR: failed to establish gRPC connection. ", err)
	}
	defer conn.Close()

	/* P4runtime Client インスタンス生成 */
	client := v1.NewP4RuntimeClient(conn)

	/* StreamChanel 確立 */
	ch, err := client.StreamChannel(context.TODO())
	if err != nil {
		log.Fatal("ERROR: failed to establish StreamChannel. ", err)
	}

	/* MasterArbitrationUpdate */
	_, err = MasterArbitrationUpdate(cntlInfo, ch)
	if err != nil {
		log.Fatal("ERROR: failed to get arbitration. ", err)
	}
	log.Printf("INFO: MasterArbitrationUpdate successfully done.")

	/* SetForwardingPipelineConfig */
	actionType := "VERIFY_AND_COMMIT"
	_, err = SetForwardingPipelineConfig(cntlInfo, actionType, client)
	if err != nil {
		log.Fatal("ERROR: failed to set forwarding pipeline config. ", err)
	}
	log.Printf("INFO: SetForwardingPipelineConfig successfully done.")

	/* P4Info および各 Entry 情報を読込み */
	p4infoText, err := ioutil.ReadFile(cntlInfo.p4infoPath)
	if err != nil {
		log.Fatal("ERROR: failed to read p4info file.", err)
	}
	var p4info config_v1.P4Info
	if err := proto.UnmarshalText(string(p4infoText), &p4info); err != nil {
		log.Fatal("ERROR: cannot unmarshal p4info.txt.", err)
	}
	log.Printf("INFO: P4Info is successfully loaded.")

	entries, err := ioutil.ReadFile(cntlInfo.runconfPath)
	if err != nil {
		log.Fatal("ERROR: cannot read file (runtime).", err)
	}
	var entryhelper myutils.EntryHelper
	if err := json.Unmarshal(entries, &entryhelper); err != nil {
		log.Fatal("ERROR: cannot unmarshal runtime.", err)
	}
	log.Printf("INFO: Entries (C/P configuration) are successfully loaded.")

	/* 更新後の Entity を含む Update を作成し，Write RPC で Entity を更新． */
	var updates []*v1.Update

	/* 更新後の Entity 生成（TableEntry) */
	for _, tableentryhelper := range entryhelper.TableEntries {
		tableentry, err := myutils.BuildTableEntry(tableentryhelper, p4info)
		if err != nil {
			log.Fatal("ERROR: cannot build table entry.", err)
		}
		entity := &v1.Entity{Entity: tableentry}
		update, err := myutils.NewUpdate("INSERT", entity)
		if err != nil {
			log.Fatal("ERROR: cannot create new update.", err)
		}
		updates = append(updates, update)
	}

	/* 更新後の ENtity 生成（PacketReplicationEngineEntry) */
	for _, multicastgroupentryhelper := range entryhelper.MulticastGroupEntries {
		multicastgroupentry, err := myutils.BuildMulticastGroupEntry(multicastgroupentryhelper)
		if err != nil {
			log.Fatal("ERROR: cannot build multicast group entry.", err)
		}
		entity := &v1.Entity{Entity: multicastgroupentry}
		update, err := myutils.NewUpdate("INSERT", entity)
		if err != nil {
			log.Fatal("ERROR: cannot create new update.", err)
		}
		updates = append(updates, update)
	}

	/* Write RPC で Entity を更新 */
	_, err = SendWriteRequest(cntlInfo, updates, "CONTINUE_ON_ERROR", client)
	if err != nil {
		log.Fatal("ERROR: failed to write entities. ", err)
	}
	log.Printf("INFO: Write has been successfully done.")

	/* VLAN 毎のトラヒックカウンタ値を取得 */
	var counter string
	var index int64
	var mod string
	fmt.Println("================ Counter Example ================")
	fmt.Println("usage: input [counter name] and [index = vlan ID]　and [mod]")
	fmt.Println("       input \"exit\" if you want to quit")
	fmt.Println("=================================================")
	for {
		fmt.Print("input counter name : ")
		fmt.Scan(&counter)
		if counter == "exit" {
			log.Print("INFO: Exit explicitly. Connection is going to down.")
			break
		}
		fmt.Print("input vlan ID      : ")
		fmt.Scan(&index)
		fmt.Print("input \"clear\" if want : ")
		fmt.Scan(&mod)

		counterentryhelper := &myutils.CounterEntryHelper{
			Counter: counter,
			Index:   index,
		}
		counterentry, err := myutils.BuildCounterEntry(counterentryhelper, p4info)
		if err != nil {
			log.Print("ERROR: cannot build CounterEntry.")
			continue
		}
		entities := make([]*v1.Entity, 0)
		entities = append(entities, &v1.Entity{Entity: counterentry})

		if mod == "clear" {

			cnt_id := counterentry.CounterEntry.GetCounterId()
			init_data := &v1.CounterData{
				ByteCount: 0,
				PacketCount: 0,
			}
			cntent := &v1.Entity_CounterEntry{
				CounterEntry: &v1.CounterEntry{
					CounterId: cnt_id,
					Index: &v1.Index{
						Index: 0,
					},
					Data: init_data,
				},
			}

			/*
			counterentry.CounterEntry.Data = &v1.CounterData{
				ByteCount: -1,
				PacketCount: -1,
			}
			*/

			upds := []*v1.Update{}
			upd, err := myutils.NewUpdate("MODIFY", &v1.Entity{Entity: cntent})
			upds = append(upds, upd)
			_, err = SendWriteRequest(cntlInfo, upds, "CONTINUE_ON_ERROR", client)
			if err != nil {
				log.Fatal("ERROR: write request error. ", err)
			}
			log.Println("INFO: Counter Entry is modified.")
			continue
		} else {

		}

		cnt_unit, err := myutils.GetCounterSpec_Unit(counter, p4info)
		if err != nil {
			log.Fatal("ERROR: cannot get counter unit.")
		}
		var unit string
		switch cnt_unit {
		case config_v1.CounterSpec_BYTES:
			unit = "bytes"
		case config_v1.CounterSpec_PACKETS:
			unit = "packets"
		case config_v1.CounterSpec_BOTH:
			unit = "bytes and packets"
		default:
			log.Fatal("ERROR: counter unit is invalid.")
		}

		rclient, err := CreateReadClient(cntlInfo, entities, client)
		if err != nil {
			log.Fatal("ERROR: cannot create read client.")
		} else {
			readresponse, err := rclient.Recv()
			if err != nil {
				log.Fatal("ERROR: cannot get read response.")
			}
			entity := readresponse.GetEntities()
			cnt := entity[0].GetCounterEntry()
			fmt.Println("VLAN-ID: ", index)
			fmt.Println("CNT NUM: ", cnt.Data.ByteCount, " ", unit)
		}
	}
	os.Exit(0)
}
