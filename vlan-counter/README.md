
# 概要

簡易 L2 スイッチに VLAN 毎のトラヒックカウンタを実装し，コントロールプレーンからカウンタ値を取得します．
  
- 参考記事１
  - [P4 で記述した簡易 L2 Switch にタグ VLAN（802.1Q）を対応させて VLAN 毎のトラヒックカウンタを実装する（準備編）](https://qiita.com/13ryuse4/items/cb83abd80712616e0799)
- 参考記事２
  - [P4 で記述した簡易 L2 Switch にタグ VLAN（802.1Q）を対応させて VLAN 毎のトラヒックカウンタを実装する（実装編）](https://qiita.com/13ryuse4/items/6f95ada4d248372603c2) 
- 参考記事３
  - [P4Runtime を用いて P4 で記述した簡易 L2 Switch のテーブルエントリ登録とトラヒックカウンタ値取得を行う（準備編）](https://qiita.com/13ryuse4/items/96ed8b31382e1fdd79f1)

# 動作確認手順

[p4-guide](https://github.com/jafingerhut/p4-guide)等を参照し，P4 開発環境が構築済みであることを前提とします．
動作確認手順は大きく下記の流れとなります．

1. P4 プログラムのコンパイル
2. 