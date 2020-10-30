# 概要（tentative）

meter を用いてトラヒック速度を制限し，かつ速度制限時にドロップしたトラヒック量をコントロールプレーンに通知する．
GTP を使う．TEID 毎にトラヒックカウント．P4 プログラムは internal UPF (L2switch) で流量制御をする想定．

やるべきこと
- meter を使ってトラヒック量制限
  - 5G UPF の URR 実装（トラヒック量制限の部分のみ）
    - key も 3GPP 準拠の仕様がいい
  - report format も 3GPP 準拠にしたい
  - トラヒック制限前は threshold は未設定 / トラヒック制限時は theshold を設定
- counter が特定の値に達したら CP に通知
  - CP は counter からの通知契機で meter に threshold 設定
  - 上記通知は digest で実施（DP 契機の digest 発出は？）
  - CP 側で独自 struct に unmarshal
- CP プログラム改良
  - meter entry 制御
    - table entry の逆引き（direct meter が table entry に紐つくため）
  - 送受信部分を goroutine で実装
    - 非同期な digest 受信に対応
  - 定期的に counter 値を監視するプログラム作成


