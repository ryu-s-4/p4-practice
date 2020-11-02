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
    - teidch（監視対象の TEID を goroutine に渡す channel）
    - cmdCh（監視対象の TEID の登録(ADD)・削除(DEL)を goroutine に渡す channel）
      - teidCh に先に渡す？
  - トラヒック量を超過した TEID に対応する URR エントリ（＋ direct meter entry）を登録
    - rptCh（トラヒック量を超過した TEID を counter 値を監視する gorouine からエントリ登録をする goroutine に通知する channel） 
  - runtime は table 毎に分けて，TEID と紐づく table entry を探すときに楽にする．
    - ちゃんとやるときは table 毎にデータベースでエントリ管理し，データベースから json 取得 / json 更新をする．
- 解説記事
  - 実装内容の概要
    - meter を使った流量制限機能（BMv2 の meter 実装の都合上，動作確認はパケットサイズの制限を使って確認）
  - meter の原理
  - DP 実装
  - CP 実装


