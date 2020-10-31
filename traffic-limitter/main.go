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

	// Traffic Monitor を行う goroutine を起動
	regCh = make(chan uint16, 10)
	delCh = make(chan uint16, 10)
	rptCh = make(chan uint16, 10)

	go MonitorTraffic() 
	go ControlMeter()

	// 監視対象 TEID を登録/削除
	/* TODO: 簡易 CLI で TEID の登録・削除 */

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

	for {
		for _, id := range teid {
			// TODO: TEID 毎に Counter 値を取得し，制限量と比較．超過していたら rptCh で通知
		}
	}
}

func ControlMeter() {
	// rptCh を待機し，TEID を受信したら URR エントリ登録＋MeterEntry 登録
	for {
		id := rptCh
		go Initializer(id)	
	}	
}

func Initializer(id uint16) {
	// トラヒック量超過をログ出力
	log.Println("INFO: Exceed the given traffic amount of TEID ", id)

	// 一定時間待機

	// トラヒック量超過した TEID のカウンタ値をゼロクリア
	/* TODO */
	log.Println("INFO: Counter of TEID ", id, " is initialized")

	// URR エントリ削除＋MeterEntry 削除
	/* TODO */
	log.Println("INFO: Entries for TEID ", id, " is deleted")

	return 
}

