# 概要（tentative）

meter を用いてトラヒック速度を制限し，かつ速度制限時にドロップしたトラヒック量をコントロールプレーンに通知する．

やるべきこと
- router.p4 の作成
  - meter でトラヒック量制限
  - digest で drop 通知（独自フォーマットのパケット）
    - CP 側で独自 struct に unmarshal
- CP プログラム改良
  - LPM 対応
  - 送受信部分を goroutine で実装
