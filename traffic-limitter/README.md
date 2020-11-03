# 概要（tentative）

meter を用いてトラヒック速度を制限し，かつ速度制限時にドロップしたトラヒック量をコントロールプレーンに通知する．
GTP を使う．TEID 毎にトラヒックカウント．P4 プログラムは internal UPF (L2switch) で流量制御をする想定．

やるべきこと
- meter を使ってトラヒック量制限
  - 5G UPF の簡易 URR 実装（トラヒック量制限の部分のみ）
  - トラヒック制限前は threshold は未設定 / トラヒック制限時は theshold を設定
- CP プログラム改良
  - direct counter entry 制御
  - direct meter entry 制御
    - table entry の逆引き（direct meter が table entry に紐つくため）
  - 定期的に counter 値を監視するプログラム作成（goroutine）
  - トラヒック量を超過した TEID に対応する direct meter entry を登録
  - runtime は table 毎に分けて，TEID と紐づく table entry を探すときに楽にする．
    - ちゃんとやるときは table 毎にデータベースでエントリ管理し，データベースから json 取得 / json 更新をする．
  - io.go や helper.go の各関数を具備した構造体変数 ControlPlaneClient を作り，MonitorTraffic にはその構造体変数のポインタを引数として渡す（ControlPlaneClient への write は原則行わない）
  - DirectMeter の INSER/DELETE はサポートされていない．TEID 毎の Traffic Counter は別途用意し，通過トラヒックについて確認が必要．
  - [TODO] Error 処理を整理．コンパイル＆デバッグ
- 解説記事
  - 実装内容の概要
    - meter を使った流量制限機能（BMv2 の meter 実装の都合上，動作確認はパケットサイズの制限を使って確認）
  - meter の原理
  - DP 実装
  - CP 実装


