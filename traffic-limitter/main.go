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

// Channels for the signaling / error-reporting
var sigCh chan string
var errCh chan error

// Control-plane and traffic control information
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

	// DB 管理を行う goroutine を起動
	sigCh = make(chan string, 10)
	errCh = make(chan error, 10)
	go DBManagement(sigCh, errCh)

	// sigCh を監視
	select {
	case msg := <-sigCh:
		log.Println("INFO: DB management has been correctly terminated.", msg)
	case errmsg := <-errCh:
		log.Fatal("ERROR: DB management has been unusually terminated.", errmsg)
	}
	os.Exit(0)
}

// DBManagement ...
func DBManagement(sigCh chan string, errCh chan error) {

	// DB との接続確立
	uri := "mongodb://127.0.0.1:27017"
	ctx := context.Background()
	db, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Println("ERROR: DB connection failed.")
		errCh <- err
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Println("ERROR: DB has terminated in correctly")
			errCh <- err
			return
		}
	}()

	// DB collection を取得
	collection := client.Database("test").Collection("test")

	// 監視対象の MAC アドレスを追加/削除
	var cmd string
	var mac string
	fmt.Println("========== Meter Regist/Delete ==========")
	fmt.Println("$ reg | del | exit  <MAC Addr. to be monitored>")
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
		fmt.Scanf("%s", &mac)
		if _, err := net.ParseMAC(mac); err != nil {
			log.Println("ERROR: invalid input as MAC addr.", err)
			continue
		}

		switch cmd {
		case "reg":

			/* check the dupulication */
			query := bson.M{"match": bson.M{"hdr.ethernet.srcAddr": mac}}
			if r := collection.FindOne(context.Background(), query); r != nil {
				fmt.Println("ERROR: the addr. is already registered.")
				continue
			}

			/* register the mac to mongoDB */
			table := "check_limit"
			match := "hdr.ethernet.srcAddr"
			teh := myutils.TableEntryHelper{
				/* action 無しの entry */
				Table:       table,
				Match:       map[string]interface{}{match: mac},
				Action_Name: "NoAction",
			}
			response, err := collection.InsertOne(context.Background(), teh)
			if err != nil {
				log.Println("ERROR: Inserting the data to DB has been failed.")
				errCh <- err
				return
			}

			/* kick the monitoring goroutine with errCh */
			go MonitorTraffic(response.InsertedID)

		case "del":

			/* delete the mac from mongoDB */
			query := bson.M{"match": bson.M{"hdr.ethernet.srcAddr": mac}}
			_, err = collection.DeleteOne(context.Background(), query)
			if err != nil {
				log.Println("ERROR: cannot find the mac addr. to be deleted.")
			}

		case "exit":
			sigCh <- "exit has been executed."
			return

		default:
			fmt.Println("ERROR: invalid input. [reg | del | exit] is only allowed to use.")
		}
	}
}

// MonitorTraffic ...
func MonitorTraffic(oid primitive.ObjectID) {

	wait_time := 10
	action_name := "limit_traffic"

	// Counter の測定単位の確認（BYTE 単位のみ許容）
	counter := "meter_cnt"
	unit, err := myutils.GetCounterSpec_Unit(counter, cp.P4Info)
	if err != nil {
		log.Fatal("ERROR: Failed to get CounterSpec.", err)
	}
	if unit != config_v1.CounterSpec_BYTES {
		log.Fatal("ERROR: Counter Unit is only allowed to be \"Bytes\".")
	}

	// トラヒックカウンタの定期監視
	for {

		// 一定時間待機
		time.Sleep(time.Second * wait_time)

		// mongoDB から監視対象のテーブルエントリ取得
		uri := "mongodb://127.0.0.1:27017"
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		db, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err != nil {
			panic(err)
		}
		defer func() {
			if err = client.Disconnect(ctx); err != nil {
				panic(err)
			}
		}()
		collection := client.Database("test").Collection("test")
		data := collection.FindOne(context.Background(), bson.D{"_id": oid})
		if data == nil {
			log.Println("INFO: TableEntry has been probably deleted from the DB.")
			break
		}
		teh := myutils.TableEntryHelper{}
		err = data.Decode(&teh)
		if err != nil {
			/* Error 処理 */
			break
		}

		// 監視対象のテーブルエントリのカウンタ値取得
		tableentry := teh.BuildTableEntry(cp.P4Info)
		directcounterentry := v1.Entity_DirectCounterEntry{
			DirectCounterEntry: &v1.DirectCounterEntry{
				TableEntry: tableentry.TableEntry,
			},
		}
		entities := []*v1.Entity{}
		entities = append(entities, directcounterentry)
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
			log.Fatal("ERROR: No entity is received from the read client.")
		}
		counter := entity[0].GetDirectCounterEntry()
		if counter == nil {
			log.Fatal("ERROR: No counter is received from the read client.")
		}
		cntdata := counter.GetData()
		if cntdata == nil {
			log.Fatal("ERROR: No counter data is received from the read client.")
		}

		// 制限容量をオーバーしていたら MeterConfig を登録し Initialization を起動
		if cntdata.ByteCount > limit {

			var updates []*v1.Update
			var update *v1.Update

			log.Println("INFO: The amount of the traffic exceeds the given volume.")

			// MeterConfig 登録
			updates = []*v1.Update{}
			directmeterentry_reg := &v1.Entity{
				Entity: &v1.Entity_DirectMeterEntry{
					DirectMeterEntry: &v1.DirectMeterEntry{
						TableEntry: tentry.TableEntry,
						Config:     mconf,
					},
				},
			}
			update = myutils.NewUpdate("MODIFY", directmeterentry_reg)
			updates = append(updates, update)
			_, err = cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
			if err != nil {
				/* Error 処理 */
			}
			log.Println("INFO: DirectMeterEntry has been successfully modified (limitter is enabled).")

			// 一定時間が経過するまで速度制限
			log.Println("INFO: Waiting for the cancellation ...")
			time.Sleep(time.Second * 10)
			log.Println("INFO: Traffic Limitation is initialized.")

			// カウンタ値をゼロクリア

			/*
				TODO: Bmv2 では Counter の Reset がサポートされていない様子．
				log.Println("INFO: Counter of TEID ", id, " is initialized")
			*/

			// MeterConfig 初期化
			updates = []*v1.Update{}
			directmeterentry_del := &v1.Entity{
				Entity: &v1.Entity_DirectMeterEntry{
					DirectMeterEntry: &v1.DirectMeterEntry{
						TableEntry:  entry.TableEntry,
						MeterConfig: &v1.MeterConfig{},
					},
				},
			}
			update = myutils.NewUpdate("MODIFY", directmeterentry_del)
			updates = append(updates, update)
			_, err := cp.SendWriteRequest(updates, "CONTINUE_ON_ERROR")
			if err != nil {
				/* ERROR 処理 */
				log.Fatal("ERROR: Failed to initialize MeterConfig.", err)
			}
			log.Println("INFO: DirectMeterEntry has been successfully initialized.")
		}
	}
}
