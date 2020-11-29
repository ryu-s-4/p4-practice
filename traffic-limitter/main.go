package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/p4-practice/traffic-limitter/myutils"

	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var sigCh chan string
var errCh chan error
var cp myutils.ControlPlaneClient
var limit int64
var meterconf *v1.MeterConfig

func main() {

	var (
		deviceid    uint64 = 0
		electionid         = &v1.Uint128{High: 0, Low: 1}
		p4infoPath  string = "./p4info.txt"
		devconfPath string = "./switching_meter.json"
		runconfPath string = "./runtime.json"
		err         error
	)

	/* コントロールプレーンを初期化 */
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
	updates := []*v1.Update{}
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

	/* Traffic 制限のための情報を設定 */
	var cir int64 = 500  // Committed information rate (units per sec)
	var cbr int64 = 150  // Committed burst size
	var pir int64 = 1000 // Peak information rate (units per sec)
	var pbr int64 = 300  // Peak burst size
	limit = 10000        // Bytes
	meterconf = &v1.MeterConfig{
		Cir:    cir,
		Cburst: cbr,
		Pir:    pir,
		Pburst: pbr,
	}

	/* DB 管理を行う goroutine を起動 */
	sigCh = make(chan string, 10)
	errCh = make(chan error, 10)
	go DBManagement(sigCh, errCh)

	/* DBmanagement の終了待機 */
	select {
	case msg := <-sigCh:
		log.Println("INFO: DB management has been correctly terminated. ", msg)
	case errmsg := <-errCh:
		log.Fatal("ERROR: DB management has been unusually terminated. ", errmsg)
	}
	os.Exit(0)
}

// DBManagement manages the DB for TableEntry.
func DBManagement(sigCh chan string, errCh chan error) {

	/* DB との接続確立 */
	uri := "mongodb://127.0.0.1:27017"
	ctx := context.Background()
	db, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Println("ERROR: failed to connect to the DB.")
		errCh <- err
		return
	}
	defer func() {
		if err = db.Disconnect(ctx); err != nil {
			log.Println("ERROR: disconnect incorrectly.")
			errCh <- err
			return
		}
	}()
	collection := db.Database("test").Collection("test")

	/* 監視対象の MAC アドレスを追加/削除 */
	var cmd string
	var mac string

	fmt.Println("========== Meter Regist/Delete ==========")
	fmt.Println(" [reg | del | exit]  <MAC Addr. to be monitored>")
	fmt.Println("   - reg : register the TEID to be monitored")
	fmt.Println("   - del : delete the TEID to be monitored")
	fmt.Println("   - exit: exit the CLI")
	fmt.Println("=========================================")
	for {

		/* 入力受付 */
		fmt.Scanf("%s", &cmd)
		if cmd != "exit" {
			fmt.Scanf("%s", &mac)
			if _, err := net.ParseMAC(mac); err != nil {
				log.Println("ERROR: invalid input as MAC addr. ", err)
				continue
			}
		}

		switch cmd {
		case "reg":

			/* 監視対象 MAC アドレス の登録が重複していないかを確認 */
			query := bson.M{"match": bson.M{"hdr.ethernet.srcAddr": mac}}
			r := collection.FindOne(context.Background(), query)
			if r.Err() == nil {
				log.Println("ERROR: the addr. is already registered.")
				continue
			}

			/* データプレーンへの Table Entry 登録 */
			table := "check_limit"
			match := "hdr.ethernet.srcAddr"
			teh := myutils.TableEntryHelper{
				Table:       table,
				Match:       map[string]interface{}{match: mac},
				Action_Name: "NoAction",
			}
			tableentry, err := teh.BuildTableEntry(cp.P4Info)
			if err != nil {
				log.Println("ERROR: failed to build table entry to be registerd.")
				continue
			}
			directmeterentry := &v1.Entity_DirectMeterEntry{
				DirectMeterEntry: &v1.DirectMeterEntry{
					TableEntry: tableentry.TableEntry,
					Config:     meterconf,
				},
			}
			updates := []*v1.Update{}
			update := myutils.NewUpdate("INSERT", &v1.Entity{Entity: tableentry})
			updates = append(updates, update)
			update = myutils.NewUpdate("MODIFY", &v1.Entity{Entity: directmeterentry})
			updates = append(updates, update)
			_, err = cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
			if err != nil {
				log.Println("ERROR: failed to insert the table entry to be registerd to DP. ")
				errCh <- err
				return
			}

			/* mongoDB への Table Entry 登録 */
			response, err := collection.InsertOne(context.Background(), teh)
			if err != nil {
				log.Println("ERROR: failed to insert the table entry to be registered to DB.")
				errCh <- err
				return
			}
			log.Println("INFO: successfully registerd the table entry.")

			/* トラヒック監視用の goroutine を起動 */
			log.Println("INFO: kick the monitoring goroutine for", mac)
			go MonitorTraffic(response.InsertedID.(primitive.ObjectID))

		case "del":

			/* データプレーンから監視対象 MAC アドレスを削除 */
			query := bson.M{"match": bson.M{"hdr.ethernet.srcAddr": mac}}
			r := collection.FindOne(context.Background(), query)
			if r.Err() != nil {
				fmt.Println("ERROR: the addr. is NOT registered.")
				continue
			}
			teh := myutils.TableEntryHelper{}
			err = r.Decode(&teh)
			if err != nil {
				log.Println("ERROR: failed to decode the table entry retrieved from DB. ")
				errCh <- err
				return
			}
			tableentry, err := teh.BuildTableEntry(cp.P4Info)
			if err != nil {
				log.Println("ERROR: failed to build table entry to be deleted. ")
				continue
			}
			updates := []*v1.Update{}
			update := myutils.NewUpdate("DELETE", &v1.Entity{Entity: tableentry})
			updates = append(updates, update)
			_, err = cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
			if err != nil {
				log.Println("ERROR: failed to delete the table entry in DP. ")
				errCh <- err
				return
			}

			/* delete the table entry from DB */
			_, err = collection.DeleteOne(context.Background(), query)
			if err != nil {
				log.Println("ERROR: cannot find the mac addr. to be deleted.")
			}
			log.Println("INFO: successfully deleted the table entry.")

		case "exit":
			sigCh <- "exit has been executed."
			return

		default:
			fmt.Println("ERROR: invalid input. [reg | del | exit] is only allowed to use.")
		}
	}
}

// MonitorTraffic monitors the traffic counter.
func MonitorTraffic(oid primitive.ObjectID) {

	/* Counter の測定単位確認（BYTE 単位のみ許容） */
	counter := "meter_cnt"
	unit, err := myutils.GetCounterSpec_Unit(counter, cp.P4Info, true)
	if err != nil {
		log.Println("ERROR: failed to get CounterSpec. ", err)
		return
	}
	if unit != config_v1.CounterSpec_BYTES {
		log.Println("ERROR: counter-unit must be \"Bytes\".")
		return
	}

	/* トラヒックカウンタの定期監視 */
	for {

		/* 一定時間待機 */
		var waiting time.Duration = 10
		time.Sleep(time.Second * waiting)

		/* mongoDB から監視対象のテーブルエントリ取得 */
		uri := "mongodb://127.0.0.1:27017"
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		db, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err != nil {
			log.Println("ERROR: failed to connect to the DB. ", err)
			return
		}
		defer func() {
			if err = db.Disconnect(ctx); err != nil {
				log.Println("ERROR: disconnect incorrectly. ", err)
				return
			}
		}()
		collection := db.Database("test").Collection("test")
		data := collection.FindOne(context.Background(), bson.M{"_id": oid})
		if data.Err() != nil {
			log.Println("INFO: table entry has been deleted from the DB.")
			break
		}
		teh := myutils.TableEntryHelper{}
		err = data.Decode(&teh)
		if err != nil {
			log.Println("ERROR: failed to decode the table entry retrieved from DB. ", err)
			return
		}

		// 監視対象のテーブルエントリのカウンタ値取得
		tableentry, err := teh.BuildTableEntry(cp.P4Info)
		if err != nil {
			log.Println("ERROR: failed to build the table entry to be monitored. ", err)
			return
		}
		directcounterentry := &v1.Entity{
			Entity: &v1.Entity_DirectCounterEntry{
				DirectCounterEntry: &v1.DirectCounterEntry{
					TableEntry: tableentry.TableEntry,
				},
			},
		}
		entities := []*v1.Entity{}
		entities = append(entities, directcounterentry)
		rclient, err := cp.CreateReadClient(entities)
		if err != nil {
			log.Println("ERROR: failed to create ReadClient. ", err)
			return
		}
		response, err := rclient.Recv()
		if err != nil {
			log.Println("ERROR: failed to receive read respose. ", err)
			return
		}
		entity := response.GetEntities()
		if entity == nil {
			log.Println("ERROR: No entity is received from the read client.")
			return
		}
		counter := entity[0].GetDirectCounterEntry()
		if counter == nil {
			log.Println("ERROR: No counter is received from the read client.")
			return
		}
		cntdata := counter.GetData()
		if cntdata == nil {
			log.Println("ERROR: No counter data is received from the read client.")
			return
		}

		/* 制限容量をオーバーしていたらトラヒック制限を有効化 */
		if cntdata.ByteCount > limit {

			var updates []*v1.Update
			var update *v1.Update

			log.Println("INFO: the amount of the traffic exceeds the given volume.")

			/* トラヒック制限を有効化 (NoAction -> limit_traffic に Action 変更) */
			teh.Action_Name = "limit_traffic"
			tableentry, err = teh.BuildTableEntry(cp.P4Info)
			if err != nil {
				log.Println("ERROR: failed to build table entry.", err)
				return
			}
			updates = []*v1.Update{}
			update = myutils.NewUpdate("MODIFY", &v1.Entity{Entity: tableentry})
			updates = append(updates, update)
			_, err = cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
			if err != nil {
				log.Println("ERROR: failed to modify the table entry (enable traffic limit).", err)
				return
			}
			log.Println("INFO: table entry has been successfully modified (limitter is enabled).")

			/* 一定時間が経過するまで速度制限 */
			log.Println("INFO: waiting for the cancellation ...")
			time.Sleep(time.Second * waiting)

			/* トラヒック制限を解除 (limit_traffic -> NoAction に Action 変更) */
			teh.Action_Name = "NoAction"
			tableentry, err = teh.BuildTableEntry(cp.P4Info)
			if err != nil {
				log.Println("ERROR: failed to build table entry. ", err)
				return
			}
			updates = []*v1.Update{}
			update = myutils.NewUpdate("MODIFY", &v1.Entity{Entity: tableentry})
			updates = append(updates, update)
			_, err = cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
			if err != nil {
				log.Println("ERROR: failed to initialize MeterConfig. ", err)
				return
			}
			log.Println("INFO: table entry has been successfully initialized (limitter is canceled).")

			/* カウンタを初期化 */
			var zero_int64 int64 = 0
			dce := &v1.Entity_DirectCounterEntry{
				DirectCounterEntry: &v1.DirectCounterEntry{
					TableEntry: tableentry.TableEntry,
					Data: &v1.CounterData{
						ByteCount:   zero_int64,
						PacketCount: zero_int64,
					},
				},
			}
			updates = []*v1.Update{}
			update = myutils.NewUpdate("MODIFY", &v1.Entity{Entity: dce})
			updates = append(updates, update)
			_, err := cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
			if err != nil {
				log.Println("ERROR: failed to clear the counter. ", err)
				return
			}
			log.Println("INFO: counter is successfully cleared.")
		}
	}
	log.Println("INFO: monitoring goroutine has been successfully terminated.")
	return
}
