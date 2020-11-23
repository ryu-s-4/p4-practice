# 概要（tentative）

meter を用いてトラヒック速度を制限し，かつ速度制限時にドロップしたトラヒック量をコントロールプレーンに通知する．
ToR スイッチを想定し，送信元 MAC アドレス毎に流量制限をかける（配下のサーバに流量制限をかけるイメージ）
制御対象外の MAC アドレスを C/P から登録できる．

やるべきこと
- meter を使ってトラヒック量制限
  - 送信元 MAC 毎にトラヒックカウント
  - トラヒック量監視@C/P を行い，閾値を超えたら流量制御用のテーブルにエントリ登録

実装改良箇所
- DP プログラム（宛先 MAC を見て L2 転送しつつ，送信元 MAC を見てトラヒックカウント） ★ 済
  - switching 用のテーブルに加えて，流量制御のテーブルを用意
    - 制限容量を超えたら meterconfig を登録し action を limit_traffic に modify（一定時間経過後に action を空にして MODIFY） 
- CP プログラム
  - 監視対象の MAC アドレスを登録する goroutine（DB management) ★ 済
    - 監視対象の MAC アドレスを登録 / 削除する CLI
    - TableEntryHelper 構造体の形式で mongoDB に登録 / から削除
    - 各 MAC 監視用の goroutine を起動
  - 流量監視用 の goroutine
    - mongoDB に登録した ID とタイマーを保持．タイマー経過後，ID で mongoDB からエントリ取得 / カウンタ値取得，を繰り返す． ★済
      -  mongoDB から取得し Helper 構造体変数に落とし込む．
      -  mongoDB から取得に失敗したら，その旨を errCh に通知して goroutine　を落とす
      -  Helpter 構造体変数から TableEntry を build して DirectCounterEntry を作成し Read RCP で取得．
    - 制限容量の超過の確認，DirectMeterConfig の登録
      - 上記で取得した DirectCounter 値を Limit（制限容量）と比較して超過検知
      - 超過していたら上記で生成した TableEntry から DirectMeterEntry を作成し Write RPC（エントリの modify）
      - 一定期間速度制限をかけて，その後解除


- 解説記事（〜11末）
  - 実装内容の概要
    - meter を使った流量制限機能（BMv2 の meter 実装の都合上，動作確認はパケットサイズの制限を使って確認）
  - meter の原理
  - DP 実装
  - CP 実装
- 解説記事２（〜11末）：備忘もかねて記事作成
  - mongoDB の概要
  - golang による mongoDB 操作の基礎

- その他 TODO
  - vlan-counter の実装解説記事
    - GO 言語によるテーブルエントリ登録，マルチキャストグループ登録，カウンタ値取得の実装
    - P4Runtime の仕様と照らし合わせながら


