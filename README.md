# 二人向けの番組推薦 API

Go 1.27.1 / Echo / GORM / MySQL。coreとadapterの二層構成です。

## 初期設定
既存の `user`・`episode`・`series` テーブルを含むデータベースを用意してください。`compose.yaml` を使う場合は、マスターダンプを `db/migrations/00_agtdb_dump.sql` に配置します。

DB接続設定は `.env.example` を参照して環境変数へ設定します（アプリは `.env` を自動読込しません）。

### 環境構築
開発環境に必要なツールはmiseでインストールできます。以下のコマンドでプロジェクトのセットアップに必要なGoとgolang-migrateがインストールされます。

```sh
mise install
```

## DB起動
以下のコマンドでdockerを使ってDBを起動できます。

```sh
mise run db-up
```

初回起動時のみ必要なテーブルの作成とデータの投入で以下のコマンドが必要です。
seed/data配下にcsvデータを配置した後に実行してください。

```sh
mise run db-migrate
mise run seed
```

詳細は[mise.toml](mise.toml)を参照してください。

## 起動

```sh
mise run dev
```

ポートは8080です。すべてのAPIで、既存ユーザーのIDを `X-User-ID` ヘッダーに指定します。この方式は課題・ローカル開発用で、本人確認を行いません。

## APIと操作順

1. `POST /rooms` でルーム名・パスワードを指定して作成。
2. 相手が `POST /rooms/join` で参加。
3. 各参加者が `GET /rooms/{room_id}/ws` へWebSocket接続。
4. `GET /rooms/{room_id}/members/{user_id}/viewing-history` で番組単位の履歴を取得。双方の履歴を参照でき、`limit`（1〜100、初期値20）と `cursor` でページング可能。
5. `PUT /rooms/{room_id}/viewing-history` へ自分の `included_series_ids` を送信。空配列で選択を解除。元の視聴履歴は変更しません。
6. 二人分がそろうと202で計算を開始し、WebSocketで `recommendation.completed` または `recommendation.failed` を通知。
7. `POST /rooms/{room_id}/preference` の `weight` で作成者寄りの重みを変更（0は相手寄り、1は作成者寄り）。
8. `DELETE /rooms/{room_id}/membership` で退出。所有者を引き継ぎ、最後の参加者が退出するとルームを削除。

詳細は [OpenAPI](docs/openapi.yaml) と [設計書](docs/system-design-organized.md) を参照してください。WebSocketは `X-User-ID` を設定できるAPIクライアントから接続します。ブラウザ標準WebSocket APIでは任意ヘッダーを指定できません。

## 推薦方式

各参加者が選択した番組のジャンル割合を作り、作成者の割合 × weight ＋ 相手の割合 × (1 − weight) で候補番組を採点します。

- 選択した番組は候補から除外します。視聴済みでも選択されていない番組は候補に残ります。
- スコアが正の番組を降順に最大10件返します。同点は番組IDの辞書順です。
- ジャンルが空の番組は加点しません。該当候補がなければ空配列で成功します。
- 重みは既存DBの精度に合わせて小数第4位へ丸めます。

## 構成
二層構造でのヘキサゴナルアーキテクチャを意識しました。core側はadapter側を知らず、Interfaceに対して依存します。
短期間での開発なので、開発速度を落とさずにできるだけ抽象化をしたいと考えたのでcoreとadapterの二つのみでパッケージを分けれる二層構成を選びました。
```text
internal/
├── room/
│   ├── core/                  # ルーム・参加・退出
│   └── adapter/{db,http}/
├── history/
│   ├── core/                  # 番組単位の履歴・ページング
│   └── adapter/{db,http}/
├── recommendation/
│   ├── core/                  # 推薦ルール・状態遷移・通知ポート
│   └── adapter/{db,http}/     # マスター／重み保存・HTTP／WebSocket
└── shared/{api,auth,config,db}/
```

## 検証

```sh
go test -race ./...
go vet ./...
go build ./...
```

テストにはSQLモックによるDBクエリ・認可検証と、ローカルHTTP/WebSocketサーバーによる操作フロー検証を含みます。MySQL実機の検証とは別です。テスト実行環境にはローカルポートの使用権限が必要です。

