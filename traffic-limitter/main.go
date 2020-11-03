/* GTP-U tunneling with traffic limitter using meter. */

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/p4-practice/traffic-limitter/myutils"

	"github.com/golang/protobuf/proto"
	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
)

// Controll channel for register / delete TEID to be monitored. 
regCh chan uint16
delCh chan uint16

cp := myutils.ControlPlaneClient{
	deviceid: 0,
	electionid: &v1.Uint128{ High: 0, Low: 1},
}

limit := 1000000 /* トラヒック量の制限値 */
mconf := &v1.MeterConfig{
	Cir: 10000,  // 10KBps = 80kbps
	Cburst: 500, // 500 Bytes
	Pir: 5000, // 5KBps = 40kbps
	Pburst 250, // 250 Bytes
}

func main() {

	/* 各種 config を初期化 */
	p4infoPath := "./p4info.txt"
	devconfPath := "./switching.json"
	runconfPath := "./runtime.json"

	err := cp.InitConfig(p4infoPath, devconfPath, runconfPath)
	if err != nil {
		log.Fatal("ERROR: failed to initialize the configurations. ", err)
	}
	log.Println("INFO: P4Info/ForwardingPipelineConfig/EntryHelper is successfully loaded.")

	/* gRPC connection 確立 */
	addr := "127.0.0.1" /* gRPC サーバのアドレス */
	port := "50051"     /* gRPC サーバのポート番号 */

	conn, err := grpc.Dial(addr+":"+port, grpc.WithInsecure())
	if err != nil {
		log.Fatal("ERROR: failed to establish gRPC connection. ", err)
	}
	defer conn.Close()

	/* P4runtime Client インスタンス生成 */
	cp.client := v1.NewP4RuntimeClient(conn)

	/* StreamChanel 確立 */
	err := cp.InitChannel()
	if err != nil {
		log.Fatal("ERROR: failed to establish StreamChannel. ", err)
	}
	log.Println("INFO: StreamChannel is successfully established.")

	/* MasterArbitrationUpdate */
	_, err := cp.MasterArbitrationUpdate()
	if err != nil {
		log.Fatal("ERROR: failed to get the arbitration. ", err)
	}
	log.Printf("INFO: MasterArbitrationUpdate successfully done.")

	/* SetForwardingPipelineConfig */
	_, err := cp.SetForwardingPipelineConfig("VERIFY_AND_COMMIT")
	if err != nil {
		log.Fatal("ERROR: failed to set forwarding pipeline config. ", err)
	}
	log.Printf("INFO: SetForwardingPipelineConfig successfully done.")

	/* Table Entry / Multicast Group Entry を登録 */
	var updates []*v1.Update
	for _, h := range cp.entries.TableEntries {
		tent, err := h.BuildTableEntry(cp.p4info)
		if err := nil {
			log.Fatal("ERROR: failed to build table entry. ", err)
		}
		update := &v1.Update{
			Type: "INSERT",
			Entity: tent,
		}
		updates = append(updates, update)
	}
	for _, h := range cp.entries.MulticastGroupEntries {
		ment, err := h.BuildMulticastGroupEntry(cp.p4info)
		if err := nil {
			log.Fatal("ERROR: failed to build multicast group entry. ", err)
		}
		update := &v1.Update{
			Type: "INSERT",
			Entity: ment,
		}
		updates = append(updates, update)		
	}
	err := cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
	if err != nil {
		log.Fatal("ERROR: failed to write entries. ", err)
	}
	log.Println("INFO: Entries are successfully written.")

	// Traffic Monitor を行う goroutine を起動
	regCh = make(chan uint16, 10)
	delCh = make(chan uint16, 10)

	go MonitorTraffic() 

	// 監視対象 TEID を登録/削除
	/* TODO: 簡易 CLI で TEID の登録・削除 */
	cmd string
	teid uint16
	fmt.Println("========== Meter Regist/Delete ==========")
	fmt.Println("$ reg | del | exit  <TEID> ")
	fmt.Println("   - reg : register the TEID to be monitored")
	fmt.Println("   - del : delete the TEID to be monitored")
	fmt.Println("   - exit: exit the CLI")
	fmt.Println("=========================================")
	for {
		fmt.Print("$")
		fmt.Scanf("%s", &cmd)
		if cmd == "exit" {
			break
		}
		fmt.Scanf("%d", &teid)

		switch cmd {
		case "reg":
			regCh <- teid
			fmt.Printf("INFO: TEID %d is registered.\n", teid)
		case "del":
			delCh <- teid
			fmt.Printf("INFO: TEID %d is deleted.\n", teid)
		default:
			fmt.Println("ERROR: invalid input. [reg | del | exit] is only allowed to use.")
		}
	}
	os.Exit(0)
}

func MonitorTraffic() {

	var teid []uint16

	// register TEID to be monitored
	go func() {
		for {
			id := <- regCh
			teid = append(teid, id)
		}
	}

	// delete TEID to be monitored
	go func {
		for {
			id := <- delCh
			for _, i := range teid {
				if (i == id) {
					if (i == 0) {
						teid = teid[1:]
					} else if (i == (len(teid) - 1)) {
						teid = teid[:(len(teid) - 1)]
					} else {
						teid = append(teid[:i], teid[i+1:]...)
					}
				}
			}
		}
	}

	key := "hdr.gtu_u.teid" /* TEID を逆引きするための key 値 */
	counter := "meter_cnt"
	unit := myutils.GetCounterSpec_Unit(counter, cp.p4info)
	if unit != config_v1.CounterSpec_BYTES{
		log.Fatal("ERROR: Counter Unit is only allowed to be \"Bytes\".")
	}

	for {
		for _, id := range teid {

			/* TEID から table entry 生成．helper から逆引き．もっとちゃんとしようとするとデータベースからエントリの json ファイル引っ張ってきて，helper 変数に落とし込んで build entry */
			var entry *v1.TableEntry = nil
			for _, h :=  range cp.entries.TableEntries {
				if h.Match[key] == id {
					entry = h.BuildTableEntry(cp.p4info)
					break
				}
			}
			if (entry == nil) {
				/* ERROR 処理 */
				/* ERROR ログを出力し id を teid から削除する．delCh <- id　する．*/
			}

			// READ RPC でカウンタ値を取得
			entities := []*v1.Entity{}
			entities = append(entities, 
				&v1.Entity{ 
					Entity: &Entity_DirectCounterEntry{ 
						DirectCounterEntry: &v1.DirectCounterEntry{
							TableEntry: entry,
						},
					},
				},
			)
			rclient := cp.CreateReadClient(entities)
			entitiy := (rclient.Recv()).GetEntities()
			if entitiy == nil {
				/* ERROR 処理 */
			}
			counter := entity[0].GetDirectCounterEntry()
			if counter == nil {
				/* ERROR 処理 */
			}

			// 取得した各カウンタ値について超過有無を確認．超過していたら MeterEntry を生成し，initialize 呼び出し（goroutine）
			cnt := counter.Data.ByteCount
			if limit < cnt {
				log.Println("INFO: Exceed the given traffic amount of TEID ", id)
				dmeterentry := &v1.Entitity{
					Entitiy: &v1.Entity_DirectMeterEntry{
						DirectMeterEntry: &v1.DirectMeterEntry{
							TableEntry: entry,
							Config: mconf,
						}
					}
				}
				update, err := myutils.NewUpdate("INSERT", dmeterentry)
				if err != nil {
					/* ERROR 処理 */
				}
				updates := []*v1.Update{update}
				_, err := cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR") 
				if err != nil {
					/* ERROR 処理 */
				}
				log.Println("INFO: Meter Entry is successfully written.")
				go Initializer(entry)
			}
		}

		// 一定時間待機
		time.Sleep(time.Second * 2)
	}
}

func Initializer(entry *v1.TableEntry) {

	// 一定時間待機（速度制限）
	log.Println("INFO: Waiting for the cancellation ... (for 10 seconds)")
	time.Sleep(time.Second * 10)
	log.Println("INFO: Traffic Limitation is cancelled.")
	
	// トラヒック量超過した TEID のカウンタ値をゼロクリア
	/* TODO: Bmv2 では Counter の Reset がサポートされていない様子．
	log.Println("INFO: Counter of TEID ", id, " is initialized")
	*/

	// MeterEntry 削除
	updates := []*v1.Update{}
	update := myutils.NewUpdate("DELETE", &v1.Entity{Entity: entry})
	err := cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
	if err != nil {
		/* ERROR 処理 */
	}
	log.Println("INFO: Entries for TEID ", id, " is deleted")

	return 
}

