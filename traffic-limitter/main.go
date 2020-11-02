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

// Controll channel for report TEID that exceeds the given amount.
rptCh chan uint16
logCh chan uint16

func main() {

	/* 各種情報を設定 */
	cntlInfo := myutils.ControllerInfo{
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
	_, err = myutils.MasterArbitrationUpdate(cntlInfo, ch)
	if err != nil {
		log.Fatal("ERROR: failed to get arbitration. ", err)
	}
	log.Printf("INFO: MasterArbitrationUpdate successfully done.")

	/* SetForwardingPipelineConfig */
	actionType := "VERIFY_AND_COMMIT"
	_, err = myutils.SetForwardingPipelineConfig(cntlInfo, actionType, client)
	if err != nil {
		log.Fatal("ERROR: failed to set forwarding pipeline config. ", err)
	}
	log.Printf("INFO: SetForwardingPipelineConfig successfully done.")

	/* P4Info を読込み */
	p4infoText, err := ioutil.ReadFile(cntlInfo.p4infoPath)
	if err != nil {
		log.Fatal("ERROR: failed to read p4info file.", err)
	}
	var p4info config_v1.P4Info
	if err := proto.UnmarshalText(string(p4infoText), &p4info); err != nil {
		log.Fatal("ERROR: cannot unmarshal p4info.txt.", err)
	}
	log.Printf("INFO: P4Info is successfully loaded.")

	/* Entry Helper を読込み */
	entries, err := ioutil.ReadFile(cntlInfo.runconfPath)
	if err != nil {
		log.Fatal("ERROR: cannot read file (runtime).", err)
	}
	var entryhelper myutils.EntryHelper
	if err := json.Unmarshal(entries, &entryhelper); err != nil {
		log.Fatal("ERROR: cannot unmarshal runtime.", err)
	}
	log.Printf("INFO: Entries (C/P configuration) are successfully loaded.")

	// Traffic Monitor を行う goroutine を起動
	regCh = make(chan uint16, 10)
	delCh = make(chan uint16, 10)
	rptCh = make(chan uint16, 10)

	go MonitorTraffic() 
	go ControlMeter()

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

	limit := 1000000 /* トラヒック量の制限値 */
	for {
		for _, id := range teid {
			// TODO: TEID 毎に Counter 値を取得し，制限量と比較．
			
			/* TEID から table entry 生成
			   helper から逆引き．もっとちゃんとしようとするとデータベースからエントリの json ファイル引っ張ってきて，helper 変数に落とし込んで build entry */
			var entry *v1.TableEntry
			for _, h :=  range entryhelper.TableEntries {
				// TEID が一致するエントリを探索．h の key 値を map[string]interface{} で返す関数を helper に作って，各 h の key をチェック
			}

			// 取得した TEID の DirectCounterEntry を生成．
		}

		// READ RPC で一気にカウンタ値を取得

		// 取得した各カウンタ値について超過有無を確認
		for _, cnt := range {
			// 超過していたら rptCh で通知．TEID を table entry から逆引きする関数用意する．
			rptCh <- /* TEID */
		}

		// 一定時間待機
		time.Sleep(time.Second * 2)
	}
}

func ControlMeter() {
	// rptCh を待機し，TEID を受信したら MeterEntry 登録（流量制御設定）
	for {
		id := <- rptCh

		// トラヒック量超過をログ出力
		log.Println("INFO: Exceed the given traffic amount of TEID ", id)

		// MeterEntry 登録

		// 初期化プロセス
		go Initializer(id)	
	}	
}

func Initializer(id uint16) {

	// 一定時間待機（速度制限）
	time.Sleep(time.Second * 5)

	// トラヒック量超過した TEID のカウンタ値をゼロクリア
	/* TODO */
	log.Println("INFO: Counter of TEID ", id, " is initialized")

	// MeterEntry 削除
	/* TODO */
	log.Println("INFO: Entries for TEID ", id, " is deleted")

	return 
}

