# 概要と解説記事

簡易L2スイッチに送信元 MAC アドレス毎のトラヒック制限機能を実装し，登録した MAC アドレス発の総トラヒック量が指定の容量をオーバーした場合に [meter]() による流量制御を適用します．また，監視対象の MAC アドレス（に対応するテーブルエントリ）を管理する DB を活用し，監視対象の  MAC アドレスを動的に登録 / 削除します．[p4-guide]() 等を参照し P4 開発環境が構築済みであることを前提とします．動作確認手順は大きく下記となります．

1. P4 プログラムのコンパイル
2. 動作確認環境の構築
3. スイッチエミュレータ（BMv2）起動
4. テーブルエントリー設定
5. mongoDB 起動
6. C/P プログラム実行
7. 監視対象の MAC アドレス登録 / 削除
8. トラヒック制限（流量制御の確認）

なお，P4 プログラムおよび C/P プログラムの中身については下記参考記事を参照ください．

- 参考記事１（執筆中）
  - [送信元 MAC アドレス毎のトラヒック制限機能を Meter(RFC2698) を用いて P4/P4Runtime で実装する（準備編）]()
- 参考記事２（執筆中）
  - [送信元 MAC アドレス毎のトラヒック制限機能を Meter(RFC2698) を用いて P4/P4Runtime で実装する（実装編）]()

# 注意事項

今回は監視対象の MAC アドレス（に対応するテーブルエントリ）を管理するために [mongoDB](https://www.mongodb.com/) を使用しています．[公式のインストールマニュアル](https://docs.mongodb.com/manual/installation/)等を参照し事前にインストールされていることを確認してください．

# 動作確認手順

本 repository を clone した後，下記のように P4 プログラムをコンパイルします．コンパイル後，カレントディレクトリに ```p4info.txt``` と ```switching_meter.json``` が生成されていることを確認してください．

```
> cd p4-practice/traffic-limitter
> p4c --std p4_16 -b bmv2 --p4runtime-files p4info.txt switching_meter.p4
> ls
p4info.txt  switching_meter.json  ...
```

続いて，動作環境用の環境を構築します．今回は下記のような構成とします．

```
                                -----
                               |host3|
                                -----
             192.168.0.0/24     .3|
                                  |                     
             -----  .1     ----------------      .5 -----
            |host1| ----- |BMv2 (P4 target)| ----- |host5|
             -----        |    L2 Switch   |        -----
                           ---------------- 
                                  |
                                .7|
                                -----
                               |host7|
                                -----
```

まず，[BMv2 の公式 repository が提供するシェルスクリプト](https://github.com/p4lang/behavioral-model/blob/master/tools/veth_setup.sh)でインターフェース設定を下記のように行います．なお ```behavioral-model``` ディレクトリの場所はインストール時のディレクトリに依存するため注意してください（[p4-guide ](https://github.com/jafingerhut/p4-guide)で環境構築を行った場合は P4-guide を clone したディレクトリと同じディレクトリに clone されているかと思います）．

```
> sudo behavioral-model/tools/veth_setup.sh
```

インターフェース設定が終わったら ```p4-practice/traffic-limitter``` ディレクトリに戻り，下記のようにシェルスクリプトを実行し BMv2 以外の部分を構築します．

```
> cd p4-practice/traffic-limitter
> sudo ./setup.sh -c 
```

下記のように BMv2 を起動すれば環境構築は完了です．

```
> sudo simple_switch_grpc --no-p4 -i 0@veth0 -i 1@veth2 -i 2@veth4 -i 3@veth6 --log-console -L trace -- --grpc-server-addr 0.0.0.0:50051
```

BMv2 にて L2 転送を行うためにテーブルエントリ設定，マルチキャストグループ設定が必要となります．別ターミナルを起動して，[前回](../vlan-counter/README.md)と同様 ```runtime.json``` に MAC アドレスを入力し，C/P プログラムがこちらの json ファイルを読み込むことでテーブルエントリ，マルチキャストグループが設定されます．
host1 への L2 転送を行うためのテーブルエントリは下記のように入力します．

```
> cd p4-practice/traffic-limitter
> sudo ip netns exec host1 ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
8: veth1@if9: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9500 qdisc noqueue state UP group default qlen 1000
    link/ether b6:5b:d1:de:07:25 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 192.168.0.1/24 scope global veth1
       valid_lft forever preferred_lft forever
    inet6 fe80::b45b:d1ff:fede:725/64 scope link 
       valid_lft forever preferred_lft forever

> vi runtime.json
# "hdr.ethernet.dstAddr" の部分を上記で確認した MAC アドレスに変更

====== runtime.json =====
{
    "table_entries" : [

        <中略>

        {
            "table": "MyIngress.mac_exact",
            "match": {
                "hdr.ethernet.dstAddr": "host1's MAC address" <- 上記で確認した MAC アドレスに変更
            },
            "action_name": "MyIngress.switching",
            "action_params": {
                "port": 0
            }
        },

        <中略>

    ]
}
```

```runtime.json``` へのテーブルエントリの書き込みが完了したら mongoDB を起動し C/P プログラムを実行します．
特に問題なく処理が進むと下記のように入力を待ち受ける状態となります．

```
> sudo systemctl start mongod
> go run main.go 
2020/12/02 18:52:51 INFO: P4Info/ForwardingPipelineConfig/EntryHelper is successfully loaded.
2020/12/02 18:52:51 INFO: StreamChannel is successfully established.
2020/12/02 18:52:51 INFO: MasterArbitrationUpdate successfully done.
2020/12/02 18:52:51 INFO: SetForwardingPipelineConfig successfully done.
2020/12/02 18:52:51 INFO: Entries are successfully written.
========== Meter Regist/Delete ==========
 [reg | del | exit]  <MAC Addr. to be monitored>
   - reg : register the TEID to be monitored
   - del : delete the TEID to be monitored
   - exit: exit the CLI
=========================================
[入力待ち]
```

```reg <MACアドレス>``` で監視対象の MAC アドレスを登録します．例えば，host1 発のトラヒックを監視対象とする場合は下記のように入力します（host1 の MAC アドレスはテーブルエントリ設定時と同様 ```ip a``` で確認します）．

```
reg  b6:5b:d1:de:07:25 [Enter]
```

無事，MAC アドレスが登録され，トラヒック監視用の goroutine が起動すると下記のようなメッセージが出力されます．

```
reg  b6:5b:d1:de:07:25 [Enter]
2020/12/02 18:57:07 INFO: successfully registerd the table entry.
2020/12/02 18:57:07 INFO: kick the monitoring goroutine for b6:5b:d1:de:07:25
[入力待ち]
```

デフォルトでは 10秒間隔でデータプレーンからカウンタ値を取得し制限容量との比較を行います．今回は peak 超過（ Meter でいう RED）の場合にトラヒック制限をかける仕様となっているため，より動作確認がしやすい peak バーストサイズによる動作確認を行います（ peak レートでもトラヒック制限がかけれますが，BMv2 にトラヒックジェネレータを接続するのが億劫だった = やり方がよく分からなかったためバーストサイズで確認します）．別ターミナルを起動して ```ping``` コマンドの ```-s``` オプションを使い host1 下記のようにトラヒックを流します．

```
> sudo ip netns exec host1 ping -s 500 192.168.0.3
PING 192.168.0.3 (192.168.0.3) 500(528) bytes of data.
508 bytes from 192.168.0.3: icmp_seq=1 ttl=64 time=2.47 ms
508 bytes from 192.168.0.3: icmp_seq=2 ttl=64 time=2.91 ms
508 bytes from 192.168.0.3: icmp_seq=3 ttl=64 time=1.90 ms
508 bytes from 192.168.0.3: icmp_seq=4 ttl=64 time=1.85 ms
508 bytes from 192.168.0.3: icmp_seq=5 ttl=64 time=8.25 ms
...
```

1秒毎に約 500 byte のトラヒックを流すため，20 秒くらいで容量制限に達します．容量制限に達すると下記のようなメッセージが出力され，ping が流れなくなります．

```
2020/12/02 18:59:27 INFO: the amount of the traffic exceeds the given volume.
2020/12/02 18:59:27 INFO: table entry has been successfully modified (limitter is enabled).
2020/12/02 18:59:27 INFO: waiting for the cancellation ...
[入力待ち]
```

上記メッセージはトラヒック監視用の goroutine が容量超過を検知し、当該 MAC アドレスに紐付くエントリの Action を NoAction から traffic_limit に変更したことを表しています．デフォルトでは 10 秒間トラヒック制限を有効化し，10秒経過後は下記のようなメッセージとともにトラヒック制限を解除（Action を traffic_limit から NoAction に再度変更）およびカウンタ値の初期化を行います．トラヒック制限解除とともに ping が再度流れるようになります．

```
2020/12/02 18:59:37 INFO: table entry has been successfully initialized (limitter is canceled).
2020/12/02 18:59:37 INFO: counter is successfully cleared.
[入力待ち]
```

監視対象の MAC アドレスを削除する場合は ```del  <MACアドレス>``` を入力します．無事，MAC アドレスが削除されると下記のようにメッセージが出力されます．

```
2020/12/02 19:03:11 INFO: successfully deleted the table entry.
[入力待ち]
```

MAC アドレスが削除されるとトラヒック監視用の goroutine は DB から情報取得に失敗し終了します．トラヒック監視用の goroutine が終了すると下記のようなメッセージが出力されます（MAC アドレスを削除した時点の，次のカウンタ値取得のタイミングで終了します）．

```
2020/12/02 19:03:17 INFO: table entry has been deleted from the DB.
2020/12/02 19:03:17 INFO: monitoring goroutine has been successfully terminated.
[入力待ち]
```

host1 等の netns や link を削除する場合は下記を実行します。

```
> sudo ./setup.sh -d
```

なお、何らかのエラーにより C/P プログラムが途中で終了すると、DB に情報が残ったまま C/P プログラムだけ終了してしまう可能性があります。DB を全てクリアしたい場合は下記を実行して DB を初期化します。

```
> go run initdb.go
```