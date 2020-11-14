# Mongo DB についてのメモ

Mongo DB を使って何をしたいか．
- runtime.json の管理（各種エントリの追加，削除）を mongoDB で実施したい
  - mongoDB の API を活用してエントリ登録，削除（エントリ登録時，削除時）
  - mongoDB の API か活用してエントリ取得（エントリ参照時）

下記手順が必要

1. DB の初期化（CLI で実施？）
2. DB への connect
3. DB への Insert/Delete
4. DB からの Read