
# 概要と参考記事

簡易 L2 スイッチに VLAN 毎のトラヒックカウンタを実装し，コントロールプレーンからカウンタ値を取得します．[p4-guide ](https://github.com/jafingerhut/p4-guide)等を参照し P4 開発環境が構築済みであることを前提とします．動作確認手順は大きく下記の流れとなります．

1. P4 プログラムのコンパイル
2. 動作確認環境の構築
3. スイッチエミュレータ（BMv2）起動
4. テーブルエントリー設定
5. C/P プログラム実行（カウンタ値取得）

なお，P4 プログラムおよび C/P プログラムの中身については下記参考記事を参照ください．

- 参考記事１
  - [P4 で記述した簡易 L2 Switch にタグ VLAN（802.1Q）を対応させて VLAN 毎のトラヒックカウンタを実装する（準備編）](https://qiita.com/13ryuse4/items/cb83abd80712616e0799)
- 参考記事２
  - [P4 で記述した簡易 L2 Switch にタグ VLAN（802.1Q）を対応させて VLAN 毎のトラヒックカウンタを実装する（実装編）](https://qiita.com/13ryuse4/items/6f95ada4d248372603c2) 
- 参考記事３
  - [P4Runtime を用いて P4 で記述した簡易 L2 Switch のテーブルエントリ登録とトラヒックカウンタ値取得を行う（準備編）](https://qiita.com/13ryuse4/items/96ed8b31382e1fdd79f1)
- 参考記事４（執筆中）
  - [P4Runtime を用いて P4 で記述した簡易 L2 Switch のテーブルエントリ登録とトラヒックカウンタ値取得を行う（実装編）]()

# 注意

今回の実装，特にテーブルエントリ登録やマルチキャストグループ登録を行う部分の実装は自作 helper 関数を使った独自実装になっています．一般には p4info.txt から各 entity の ID 等を参照しつつ処理を行う必要があるためご注意ください．テーブル名等から ID を引っ張ってきて適切なデータ構造に変換する処理等の自作実装部分は ```myutils/helper.go``` を参照ください（例えば，テーブルエントリを生成する関数は ```BuildTableEntry``` という関数です）．

# 動作確認手順

本 repository を clone した後，下記のように P4 プログラムをコンパイルします．コンパイル後，カレントディレクトリに ```p4info.txt``` と ```switching.json``` が生成されていることを確認してください．

```
> cd p4-practice/vlan-counter
> p4c --std p4_16 -b bmv2 --p4runtime-files p4info.txt switching.p4
> ls
p4info.txt  switching.json  ...
```

続いて，動作環境用の環境を構築します．今回は下記のような構成とします．

```
Def. VLAN : 192.168.0.0/24                                 -----
VLAN 100  : 192.168.100.0/24                              |host3|
  -> host1, host5                                          -----
VLAN 200  : 192.168.200.0/24                               .3|
  -> host1, host3, host7                                     |                     
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

インターフェース設定が終わったら下記のようにシェルスクリプトを実行し BMv2 以外の部分を構築します．

```
> sudo ./setup_env.sh
```

下記のように BMv2 を起動すれば環境構築は完了です．

```
> sudo simple_switch_grpc --no-p4 -i 0@veth0 -i 1@veth2 -i 2@veth4 -i 3@veth6 --log-console -L trace --grpc-server-addr 0.0.0.0:50051
```

BMv2 にて L2 転送を行うためにはテーブルエントリやマルチキャストグループの登録が必要になります．今回はエントリー登録用の json ファイル（```runtime.json```）を C/P プログラムが読み込んで各種エントリーの登録を行う実装としています．各 host の MAC アドレスを確認しつつ，エントリー登録用の ```runtime.json``` を下記のように編集します（下記には host1 のデフォルト VLAN, VLAN 100 用の MAC テーブルエントリの設定方法を記載しています）．

```
> sudo ip netns exec host1 ip a
# host1 の MAC アドレスを確認

> vi runtime.json
# "EDIT" の部分を上記で確認した MAC アドレスに変更

====== runtime.json =====
{
    "table_entries" : [

        <中略>

        {
            "table": "MyIngress.mac_exact",
            "match": {
                "hdr.ethernet.dstAddr": "EDIT"
            },
            "action_name": "MyIngress.switching",
            "action_params": {
                "port": 0  <- host1 に対応
            }
        },

        <中略>

        {  
            "table": "MyIngress.mac_vlan_exact",
            "match": {    
                "hdr.vlan.id": 100,    
                "hdr.ethernet.dstAddr": "EDIT"    
            },    
            "action_name": "MyIngress.switching_vlan",    
            "action_params": {    
                "port": 0 <- host1 に対応
            }    
        },

        <中略>
    ]
}
=====
```

なお，マルチキャストグループの設定は下記で記述しています．今回は VLAN 毎に所属する host を固定しているため特に編集は不要ですが，P4 におけるマルチキャストグループは ```group ID``` 毎に ```Replica``` という Entity が出力先ポート数分だけ紐付いて管理されており，各 ```Replica``` には ```egress port（出力ポート）``` と ```instance(egress port の組を管理する ID)``` が設定されています．例えば，VLAN 100 に対応するマルチキャストグループは下記のように設定しています．

```
{
    <中略>

    "multicast_group_entries" : [
        {
            "multicast_group_id" : 2,
            "replicas" : [    
                {   
                    "egress_port" : 0,
                    "instance" : 2
                },   
                {   
                    "egress_port" : 2,    
                    "instance" : 2
                }   
            ]    
        },

        <中略> 
    ]
```

以上で C/P プログラムを実行する準備は完了です．後は下記のように C/P プログラムを実行し簡易 CLI に従ってカウンタ値を取得します．

```
> go run main.go




```

今回は ```traffic_cnt``` という名前で counter を定義しています．そのため，例えば VLAN 100 のトラヒックカウンタ値を取得したい場合は下記のように入力します．なお，終了したい場合は ```exit``` を入力すると終了します．

```
input counter name: traffic_cnt [Enter]
input vlan ID     : 100
# 結果が出力されます
```

# 後始末

TODO
