# 概要

本 repository では P4 学習用に作成した各種コードを公開します．

- vlan-counter : 簡易 L2 スイッチに VLAN 毎のトラヒックカウンタを実装し，コントロールプレーンからカウンタ値を取得します．
  - 参考 : [P4 で記述した簡易 L2 Switch にタグ VLAN（802.1Q）を対応させて VLAN 毎のトラヒックカウンタを実装する（準備編）](https://qiita.com/13ryuse4/items/cb83abd80712616e0799)
  - 参考 : [P4 で記述した簡易 L2 Switch にタグ VLAN（802.1Q）を対応させて VLAN 毎のトラヒックカウンタを実装する（実装編）](https://qiita.com/13ryuse4/items/6f95ada4d248372603c2)
  - 参考 : [P4Runtime を用いて P4 で記述した簡易 L2 Switch のテーブルエントリ登録とトラヒックカウンタ値取得を行う（準備編）](https://qiita.com/13ryuse4/items/96ed8b31382e1fdd79f1)

# 各種リンク

P4 関連で参考になる repository を下記に記載します．

- [p4-guide](https://github.com/jafingerhut/p4-guide) : 環境構築用のツールや各種サンプルコード等が充実しています．
- [tutorial](https://github.com/p4lang/tutorials) : P4.org 公式の tutorial です．
- [p4-learning](https://github.com/nsg-ethz/p4-learning) : 面白いサンプルコードが色々あります．
- to be added...

P4 関連のコミュニティを下記に記載します．

- [P4.org](https://p4.org/) : P4 の発展を主導するコミュニティです．言語仕様やインターフェース仕様（P4Runtime）等の策定を行っています．
  - [P4 language specification](https://p4.org/p4-spec/docs/P4-16-v1.2.1.html) : P4 言語仕様 v1.2.1（2020.10.20 現在）
  - [P4Runtime specification](https://p4.org/p4runtime/spec/v1.2.0/P4Runtime-Spec.html) : P4Runtime 仕様 v1.2.0（2020.10.20 現在） 
  - [PSA architecture](https://p4.org/p4-spec/docs/PSA-v1.1.0.html) : P4 target のモデルアーキティクチャ v1.1（2020.10.20 現在）
- [日本 P4 ユーザ会](https://p4users.org/) : 日本の P4 ユーザが集う会です．
