package main

import (
	"context"
	"log"

	v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
)

func main() {
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

	// StreamChanel 確立
	ch, err := client.StreamChannel(context.TODO())

	// Arbitration 処理（MasterArbitrationUpdate)

	// Write Request で複数の VLAN-ID についてカウンタ値取得

	// カウンタ値表示
}
