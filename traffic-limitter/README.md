# 概要（tentative）

meter を用いてトラヒック速度を制限し，かつ速度制限時にドロップしたトラヒック量をコントロールプレーンに通知する．
ToR スイッチを想定し，送信元 MAC アドレス毎に流量制限をかける（配下のサーバに流量制限をかけるイメージ）
但し，制御対象外の MAC アドレスを C/P から登録できる．

やるべきこと
- meter を使ってトラヒック量制限
  - 送信元 MAC 毎にトラヒックカウント
  - トラヒック量監視@C/P を行い，閾値を超えたら流量制御用のテーブルにエントリ登録

実装改良箇所
- DP プログラム（宛先 MAC を見て L2 転送しつつ，送信元 MAC を見てトラヒックカウント）
  - switching 用のテーブルに加えて，流量制御のテーブルを用意（action は全て check_traffic）
    - 制限容量を超えたら meterconfig を登録（一定時間経過後に meterconfig reset = nil を入れて MODIFY） 
- CP プログラム
  - テーブルエントリを登録する CLI 作成
    - key, action, action_param を入力
    - エントリ登録用の channel に IN
  - テーブルエントリ登録（エントリ登録用の chennel を待ち受け）する goroutine
    - key と各種情報（action, action_param）を紐つけ（Helper 構造体作成）
    - Helper 構造体を mongoDB に登録
    - エントリ登録（Write RPC） 
  - 流量監視用 の goroutine
    - 定期的に各テーブルエントリの DirectCounter を取得 
      -  mongoDB から取得し Helper 構造体変数に落とし込む．
      -  Helpter 構造体変数から TableEntry を build して DirectCounterEntry を作成し Read RCP で取得．
    - 制限容量の超過の確認，DirectMeterConfig の登録
      - 上記で取得した DirectCounter 値を Limit（制限容量）と比較して超過検知
      - 超過していたら上記で生成した TableEntry から DirectMeterEntry を作成し Write RPC（エントリの modify）
      - 初期化用の goroutine にこの TableEntry を渡す．
  - DirectMeterEntry 初期化用の goroutine
    - 一定時間待機（真剣によるときは終了時刻も渡して，グローバル時刻と比較）
    - DirectMeterEntry を初期化．MeterConfig を nil にして Write RCP（エントリの modify）　← ちゃんと動くか分からん．


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


