package main

import (
	"fmt"
	"log"
	"srv6/myutils"
	"time"

	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
)

func controller(addr string, port_num string, name string, main_channel chan string) {

	var cp myutils.ControlPlaneClient

	var (
		deviceid    uint64 = 0
		electionid         = &v1.Uint128{High: 0, Low: 1}
		p4infoPath  string = "./p4info.txt"
		devconfPath string = "./main.json"
		runconfPath string = "./config-" + name + ".json"
		err         error
	)

	/* コントロールプレーンを初期化 */
	cp.DeviceId = deviceid
	cp.ElectionId = electionid

	err = cp.InitConfig(p4infoPath, devconfPath, runconfPath)
	if err != nil {
		log.Fatal("ERROR: failed to initialize the configurations. ", err)
	}
	log.Printf("INFO[%s]: P4Info/ForwardingPipelineConfig/EntryHelper is successfully loaded.", name)

	/* gRPC connection 確立 */
	conn, err := grpc.Dial(addr+":"+port_num, grpc.WithInsecure())
	if err != nil {
		log.Fatal("ERROR: failed to establish gRPC connection. ", err)
	}
	defer conn.Close()

	/* P4runtime Client インスタンス生成 */
	cp.Client = v1.NewP4RuntimeClient(conn)

	/* StreamChanel 確立 */
	err = cp.InitChannel()
	if err != nil {
		log.Fatal("ERROR: failed to establish StreamChannel. ", err)
	}
	log.Printf("INFO[%s]: StreamChannel is successfully established.", name)

	/* MasterArbitrationUpdate */
	_, err = cp.MasterArbitrationUpdate()
	if err != nil {
		log.Fatal("ERROR: failed to get the arbitration. ", err)
	}
	log.Printf("INFO[%s]: MasterArbitrationUpdate successfully done.", name)

	/* SetForwardingPipelineConfig */
	_, err = cp.SetForwardingPipelineConfig("VERIFY_AND_COMMIT")
	if err != nil {
		log.Fatal("ERROR: failed to set forwarding pipeline config. ", err)
	}
	log.Printf("INFO[%s]: SetForwardingPipelineConfig successfully done.", name)

	/* WriteTableEntry */
	updates := []*v1.Update{}
	need_to_write := false
	for _, h := range cp.Entries.TableEntries {
		tent, err := h.BuildTableEntry(cp.P4Info)
		if err != nil {
			log.Fatal("ERROR: failed to build table entry. ", err)
		}
		update := myutils.NewUpdate("INSERT", &v1.Entity{Entity: tent})
		updates = append(updates, update)
		need_to_write = true
	}
	if need_to_write {
		_, err = cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
		if err != nil {
			log.Fatal("ERROR: failed to write entries. ", err)
		}
		log.Printf("INFO[%s]: Table Entries are successfully written.", name)
	}

	/* Write MulticastGroupEntry */
	updates = []*v1.Update{}
	need_to_write = false
	for _, h := range cp.Entries.MulticastGroupEntries {
		ment, err := h.BuildMulticastGroupEntry()
		if err != nil {
			log.Fatal("ERROR: failed to build multicast group entry. ", err)
		}
		update := myutils.NewUpdate("INSERT", &v1.Entity{Entity: ment})
		updates = append(updates, update)
		need_to_write = true

	}
	if need_to_write {
		_, err = cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
		if err != nil {
			log.Fatal("ERROR: failed to write entries. ", err)
		}
		log.Printf("INFO[%s]: MulticastGroup Entries are successfully written.", name)
	}

	/* Waiting for packet in */
	pbuf := make(chan interface{}, 10)
	ebuf := make(chan error, 10)
	go func() {
		for {
			pkt, err := cp.Channel.Recv()
			log.Printf("INFO[%s]: Something received", name)
			pbuf <- pkt
			ebuf <- err
		}
	}()

	is_loop := true
	for {
		select {

		case p := <-pbuf:

			pkt := p.(*v1.StreamMessageResponse)
			upd := pkt.GetUpdate()
			switch upd.(type) {
			case *v1.StreamMessageResponse_Packet:
				packet := pkt.GetPacket()
				log.Printf("INFO[%s]: Packet has been received %d bytes", name, len(packet.Payload))
				/* TODO: MAC learning and/or ping response */
			default:
				log.Printf("INFO[%s]: Something has been received but not packet...", name)
			}

		case e := <-ebuf:
			if e != nil {
				log.Printf("ERROR[%s]: Exit with error. %v", name, e)
				is_loop = false
				break
			}

		case _, cntn := <-main_channel:
			if !cntn {
				log.Printf("INFO[%s]: is going to down", name)
				is_loop = false
				break
			}
		default:
			/* Nothing to do */
			time.Sleep(time.Second * 1)
		}
		if !is_loop {
			break
		}
	}
}

func main() {

	main_channel := make(chan string)

	/* Initialize each device */
	go controller("127.0.0.1", "50051", "RT1", main_channel)
	time.Sleep(time.Second * 3)

	go controller("127.0.0.1", "50052", "RT2", main_channel)
	time.Sleep(time.Second * 3)

	go controller("127.0.0.1", "50053", "RT3", main_channel)
	time.Sleep(time.Second * 3)

	go controller("127.0.0.1", "50054", "RT4", main_channel)
	time.Sleep(time.Second * 3)

	/* Waiting until explicitly terminated */
	log.Println("============================================================")
	log.Println(" type \"quit[Enter]\" if you want to terminate this program ")
	log.Println("============================================================")
	for {
		var input string
		fmt.Scan(&input)
		if input == "quit" {
			break
		}
	}

	/* shutdown */
	log.Println("INFO: Shutdown (wait for a few seconds)")
	close(main_channel)
	time.Sleep(time.Second * 3)
}
