/* GTP-U tunneling with traffic limitter using meter. */

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/p4-practice/traffic-limitter/myutils"

	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
)

// Controll channel for register / delete TEID to be monitored.
var regCh chan int64
var delCh chan int64

var cp myutils.ControlPlaneClient

var limit int64
var mconf *v1.MeterConfig

func main() {

	var (
		deviceid    uint64 = 0
		electionid         = &v1.Uint128{High: 0, Low: 1}
		p4infoPath  string = "./p4info.txt"
		devconfPath string = "./gtptunnel_meter.json"
		runconfPath string = "./runtime.json"
		err         error
	)

	/* コントロールプロセスを初期化 */
	var cp myutils.ControlPlaneClient
	cp.DeviceId = deviceid
	cp.ElectionId = electionid

	err = cp.InitConfig(p4infoPath, devconfPath, runconfPath)
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
	cp.Client = v1.NewP4RuntimeClient(conn)

	/* StreamChanel 確立 */
	err = cp.InitChannel()
	if err != nil {
		log.Fatal("ERROR: failed to establish StreamChannel. ", err)
	}
	log.Println("INFO: StreamChannel is successfully established.")

	/* MasterArbitrationUpdate */
	_, err = cp.MasterArbitrationUpdate()
	if err != nil {
		log.Fatal("ERROR: failed to get the arbitration. ", err)
	}
	log.Printf("INFO: MasterArbitrationUpdate successfully done.")

	/* SetForwardingPipelineConfig */
	_, err = cp.SetForwardingPipelineConfig("VERIFY_AND_COMMIT")
	if err != nil {
		log.Fatal("ERROR: failed to set forwarding pipeline config. ", err)
	}
	log.Printf("INFO: SetForwardingPipelineConfig successfully done.")

	/* Table Entry / Multicast Group Entry を登録 */
	var updates []*v1.Update
	for _, h := range cp.Entries.TableEntries {
		tent, err := h.BuildTableEntry(cp.P4Info)
		if err != nil {
			log.Fatal("ERROR: failed to build table entry. ", err)
		}
		update := myutils.NewUpdate("INSERT", &v1.Entity{Entity: tent})
		updates = append(updates, update)
	}
	for _, h := range cp.Entries.MulticastGroupEntries {
		ment, err := h.BuildMulticastGroupEntry()
		if err != nil {
			log.Fatal("ERROR: failed to build multicast group entry. ", err)
		}
		update := myutils.NewUpdate("INSERT", &v1.Entity{Entity: ment})
		updates = append(updates, update)
	}
	_, err = cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
	if err != nil {
		log.Fatal("ERROR: failed to write entries. ", err)
	}
	log.Println("INFO: Entries are successfully written.")

	// トラヒック制限に関する情報を登録
	limit = 10000 // bytes
	mconf = &v1.MeterConfig{
		Cir:    10000, // 10KBps = 80kbps
		Cburst: 500,   // 500 Bytes
		Pir:    5000,  // 5KBps = 40kbps
		Pburst: 250,   // 250 Bytes
	}

	// Traffic Monitor を行う goroutine を起動
	regCh = make(chan int64, 10)
	delCh = make(chan int64, 10)

	go MonitorTraffic()

	// 監視対象 TEID を登録/削除
	var cmd string
	var teid int64
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

	var teid []int64

	// register TEID to be monitored
	go func() {
		for {
			id := <-regCh
			teid = append(teid, id)
		}
	}()

	// delete TEID to be monitored
	go func() {
		for {
			id_int64 := <-delCh
			id := int(id_int64)
			for _, i_int64 := range teid {
				i := int(i_int64)
				if i == id {
					if i == 0 {
						teid = teid[1:]
					} else if i == (len(teid) - 1) {
						teid = teid[:(len(teid) - 1)]
					} else {
						teid = append(teid[:i], teid[i+1:]...)
					}
				}
			}
		}
	}()

	table := "urr_exact"
	match := "hdr.gtu_u.teid"
	action_name := "limit_traffic"
	counter := "meter_cnt"
	unit, err := myutils.GetCounterSpec_Unit(counter, cp.P4Info)
	if err != nil {
		log.Fatal("ERROR: Failed to get CounterSpec.", err)
	}
	if unit != config_v1.CounterSpec_BYTES {
		log.Fatal("ERROR: Counter Unit is only allowed to be \"Bytes\".")
	}
	for {
		for _, id := range teid {

			// TEID = id のカウンタ値を取得
			cntentryhelper := myutils.CounterEntryHelper{
				Counter: counter,
				Index:   id,
			}
			cntentry, err := cntentryhelper.BuildCounterEntry(cp.P4Info)
			if err != nil {
				log.Fatal("ERROR: Counter Entry is NOT found.", err)
			}

			/* TEID から table entry 生成．helper から逆引き．
			もっとちゃんとしようとするとデータベースからエントリの json ファイル引っ張ってきて，helper 変数に落とし込んで build entry
			var entry *v1.TableEntry = nil
			for _, h :=  range cp.entries.TableEntries {
				if h.Match[key] == id {
					entry = h.BuildTableEntry(cp.p4info)
					break
				}
			}
			if (entry == nil) {
				log.Println("ERROR: TEID ", id, " is NOT included in the table entry.")
				log.Println("INFO: TEID ", id, " is deleted from teid.")
				delCh <- id
				continue
			}
			*/

			// READ RPC でカウンタ値を取得
			entities := []*v1.Entity{}
			entities = append(entities, &v1.Entity{Entity: cntentry})
			rclient, err := cp.CreateReadClient(entities)
			if err != nil {
				log.Fatal("ERROR: Failed to create ReadClient.", err)
			}
			response, err := rclient.Recv()
			if err != nil {
				log.Fatal("ERROR: Failed to receive read respose.", err)
			}
			entity := response.GetEntities()
			if entity == nil {
				log.Println("ERROR: No entity is received from the read client.")
				continue
			}
			counter := entity[0].GetDirectCounterEntry()
			if counter == nil {
				log.Println("ERROR: No counter is received from the read client.")
				continue
			}

			// 取得した各カウンタ値について超過有無を確認．超過していたら MeterEntry を生成し，initialize 呼び出し（goroutine）
			cnt := counter.Data.ByteCount
			if limit < cnt {
				log.Println("INFO: Exceed the given traffic amount of TEID ", id)

				// table entry の生成
				tentryhelper := myutils.TableEntryHelper{
					Table:       table,
					Match:       map[string]interface{}{match: id},
					Action_Name: action_name,
				}
				tentry, err := tentryhelper.BuildTableEntry(cp.P4Info)
				if err != nil {
					log.Fatal("ERROR: Cannot build the table entry.", err)
				}
				tentry_update := myutils.NewUpdate("INSERT", &v1.Entity{Entity: tentry})

				// direct meter entry の生成
				dmeterentry := &v1.Entity{
					Entity: &v1.Entity_DirectMeterEntry{
						DirectMeterEntry: &v1.DirectMeterEntry{
							TableEntry: tentry.TableEntry,
							Config:     mconf,
						},
					},
				}
				dmeter_update := myutils.NewUpdate("MODIFY", dmeterentry)

				// WRITE RPC
				updates := []*v1.Update{tentry_update, dmeter_update}
				_, err = cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
				if err != nil {
					log.Fatal("ERROR: write RPC has been failed.", err)
				}
				log.Println("INFO: TableEntry and DirectMeterEntry are successfully written.")
				go Initializer(tentry)
			}
		}

		// 一定時間待機
		time.Sleep(time.Second * 2)
	}
}

func Initializer(entry *v1.Entity_TableEntry) {

	// 一定時間待機（速度制限）
	log.Println("INFO: Waiting for the cancellation ... (for 10 seconds)")
	time.Sleep(time.Second * 10)
	log.Println("INFO: Traffic Limitation is cancelled.")

	// トラヒック量超過した TEID のカウンタ値をゼロクリア
	/* TODO: Bmv2 では Counter の Reset がサポートされていない様子．
	log.Println("INFO: Counter of TEID ", id, " is initialized")
	*/

	// Table Entry (and the corresponding DirectMeterEntry) 削除
	updates := []*v1.Update{}
	update := myutils.NewUpdate("DELETE", &v1.Entity{Entity: entry})
	updates = append(updates, update)
	_, err := cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
	if err != nil {
		/* ERROR 処理 */
		log.Fatal("ERROR: Failed to delete TableEntry.", err)
	}
	log.Println("INFO: DirectMeterEntry is successfully deleted. (traffic volume is restored)")

	return
}
