# Mongo DB についてのメモ

Mongo DB を使って何をしたいか．
- runtime.json の管理（各種エントリの追加，削除）を mongoDB で実施したい
  - mongoDB の API を活用してエントリ登録，削除（エントリ登録時，削除時）
  - mongoDB の API を活用してエントリ取得（エントリ参照時）

下記手順が必要

1. DB の初期化（CLI で実施？）
2. DB への connect
    - "mongo  mongodb://127.0.0.1:27017" で接続（IP, Port は /etc/mongodb.conf で確認，golang でどのように記述するかは要確認）
3. DB への Insert/Delete
    - 各エントリのヘッダに "entryType" : ... を付与（エントリ探索を容易化）
      - "tableentry", "multicastgroupentry", "counterentry", ...
    - db.iventory.insertOne{} / db.inventory.insertMany{} を利用（golang でどのように記述するかは要確認）
    - golang で下記を実施するための API 整理
      - collection.InsertOne()
      - collection.InsertMany()
      - collection.DeleteOne()
      - collection.DeleteMany()
4. DB からの Read
    - db.inventory.find{} を利用（golang でどのように記述するかは要確認）
    - golang で下記を実施するための API 整理
      - collection.Find()
        - query（1 condition）を指定して find する方法
        - query（複数 conditions）を指定して find する方法
        - bson.M を nest することで query を nest できる
