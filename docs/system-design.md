# 二人の視聴履歴からおすすめの番組をレコメンドする機能

## 概要
二人のユーザーの視聴履歴から、お互いが楽しめる番組をレコメンドするシステム。
スライダーみたいなので、相手寄りか自分寄りか選べてそれによって推薦度も変わる。
推薦に含める番組を選択できて、相手寄りか自分寄りか選べてそれによって推薦度も変わる。
ルームみたいなのを作って、そこに招待するイメージ。

## ユースケース
- 二人向けの推薦を取得する。,自分の視聴履歴から特定の番組を除外して、推薦を再取得する。
  -  推薦を取得したい、履歴を送信する。
- 二人の視聴履歴を取得する。
- 二人の重み変更

- ルームを作成する。
- 招待を受けてルームに参加する。
- ルームの状態・参加者を確認する。 <- websocket
- ルームから退出する。

room入室 -> 推薦に使う番組を選択 ->送信 ->　結果が揃ったら出力 -> 好みの変更や、重みづけを変更 -> 送信

## API design
### 共通エラー仕様

失敗時は、HTTPステータスと以下の形式のJSONを返す。

```json
{
  "error": {
    "code": "INVALID_WEIGHT",
    "message": "weightは0以上1以下の数値で指定してください。"
  }
}
```

- `code`: クライアントがエラー処理を分岐するための識別子。
- `message`: 人が読むための説明文。クライアントの処理分岐には使用しない。
- 内部エラーの詳細やスタックトレースはレスポンスに含めない。

#### 共通エラーコード

| HTTPステータス | エラーコード | 状況 |
|---|---|---|
| `400` | `INVALID_REQUEST` | JSON不正、重みが範囲外など |
| `401` | `UNAUTHENTICATED` | `X-User-ID`が未指定・空、またはユーザーが存在しない |
| `404` | `ROOM_NOT_FOUND` | ルームが存在しない、または非参加者 |
| `409` | `ROOM_NOT_READY` | 二人そろっていない |
| `415` | `UNSUPPORTED_MEDIA_TYPE` | JSON以外の対応していない形式 |
| `500` | `INTERNAL_ERROR` | 予期しないサーバーエラー |

### 簡易ユーザー識別

このバックエンド課題ではログインAPIやセッション管理を実装せず、すべてのHTTPリクエストで`X-User-ID`ヘッダーを必須とする。

```http
X-User-ID: user_1
```

- 値には既存の`user.id`を指定する。
- 共通ミドルウェアは値をトリムし、DBにユーザーが存在することを確認して、検証済みIDをリクエストコンテキストへ格納する。
- ヘッダーが未指定・空、またはユーザーが存在しない場合は`401 UNAUTHENTICATED`を返す。
- ハンドラーは本文やURLの値を実行ユーザーとして信用せず、コンテキストのIDを利用する。
- ユーザー識別後、ルーム操作ごとに参加資格を確認する。ルーム不存在と非参加者はどちらも`404 ROOM_NOT_FOUND`とする。
- WebSocketでもHTTPハンドシェイク時に同じヘッダーを指定する。

この方式はヘッダーの送信者が本人であることを証明せず、任意のユーザーになりすませる。そのため課題・ローカル開発専用とし、本番環境ではセッションまたは署名付きトークンへ置き換える。

ブラウザ標準のWebSocket APIは任意のヘッダーを設定できないため、今回のWebSocket接続はAPIクライアント等によるバックエンド単体利用を前提とする。ブラウザUIを接続する段階でCookieベースのセッション等へ変更する。

## 二人向けの推薦を取得する。
### 自分の視聴履歴から特定の番組を除外して、推薦を再取得する。
### 推薦を取得したい、履歴を送信する。 

#### エンドポイント
```/rooms/{room_id}/viewing-history```

#### メソッド
PUT

#### Request

```json
{
  "included_series_ids": [12, 34, 56]
}
```

#### Response

```json
{
  "status": "waiting_for_partner"
}
```

```json
{
  "status": "processing"
}
```
```
```


#### エラーコード
| HTTPステータス | コード例 | 条件 |
|---|---|---|
| 400 | `INVALID_REQUEST` | JSONの形式やIDの型が不正 |
| 403 | `NOT_ROOM_MEMBER` | ルームの参加者ではない |
| 404 | `ROOM_NOT_FOUND` | ルームが存在しない |
| 422 | `INVALID_SERIES_SELECTION` | 選択上限を超える、選択不可の番組を含む |
| 409 | `RECOMMENDATION_IN_PROGRESS` | 計算中の変更を禁止する場合 |

#### 悩み所
postしてからgetで表示させる？dbに保存するほどのものでもない気がする。できればメモリ上で処理したほうが早そう。
長いパラメーターは送りたくない。queryメソッドにする？時間あればredisとかキャッシュに積みたいね。メンターに相談したいかも,cursorペジネーションやりたい。<- memory上でやるなら話は変わってくるか。検討<- ペじネーションいらないかも<- とりあえず固定10件返すでいいや。時間あったらペ自ネーションとか考えるか。毎回計算するの重いからキャッシュとか考えたい。推薦で使うために選んだ視聴履歴をどこに保存しようね。メモリ？DB?
error codeの設計どうやってるか聞いてみたい。
ちょっと考え直しが必要かも、推薦に使う番組送信->結果発表で計算の流れのほうが簡単？
同時に送れないかね？


### 二人の好みの重みを変更して、送信する。
#### エンドポイント
```/rooms/{room_id}/preference```

#### メソッド
POST
#### Request
```json
{
  weight: 0.4
}
```

#### Response
```json
{
  status: SUCCESS,
  weight: 0.4
}
```

#### エラーコード
| HTTPステータス | コード | 発生条件 |
| --- | --- | --- |
| 400 | INVALID_WEIGHT | weightが数値だが、0以上1以下の範囲外の場合 |
| 403 | `NOT_ROOM_MEMBER` | ルームの参加者ではない |
| 404 | `ROOM_NOT_FOUND` | ルームが存在しない |


### 二人の視聴履歴を取得する。
#### エンドポイント
```/rooms/{room_id}/members/{user_id}/viewing-history```

#### メソッド
Get

#### Response
```json
{
  "items": [
    {
      "series_id": "series_c",
      "series_title": "シリーズC"
    }
  ],
  "pagination": {
    "next_cursor": "opaque_cursor",
    "has_more": true
  }
}
```

#### エラーコード
共通エラーコードに加え、以下のエラーを返す。
| HTTPステータス | コード | 発生条件 |
|---|---|---|
| `400` | `INVALID_REQUEST` | `limit`が不正、またはカーソルが不正・対象と不一致 |
| `404` | `NOT_FOUND` | ルームが存在しない、実行者が非参加者、または対象ユーザーがそのルームの参加者ではない |


## Room関連
### ルームを作成する。
#### エンドポイント
```/rooms```

#### メソッド
POST

#### Request
```json
{
  "room_name": "aaa",
  "password": "bbb"
}
```

#### Response
```json
{
  "room_id": room_123,
  "status": "waiting",
}
```

#### エラーコード
共通エラーコードに加え、以下のエラーを返す。
| HTTPステータス | コード | 発生条件 |
|---|---|---|
| 400 | `INVALID_ROOM_NAME` | ルーム名が制約を満たさない |
| 400 | `INVALID_PASSWORD` | パスワードが制約を満たさない |
| 404 | `USER_NOT_FOUND` | 指定したユーザーが存在しない |
| 409 | `ROOM_NAME_ALREADY_EXISTS` | 同じ名前のルームが存在する |


### 招待を受けてルームに参加する。

#### エンドポイント
```/rooms/join```

#### メソッド
POST

#### Request
```json
{
  "room_name": "aaa",
  "password": "bbb"
}
```

#### Response
```json
{
  "room_id": room_123,
  "status": "ready"
}
```

#### エラーコード
共通エラーコードに加え、以下のエラーを返す。
| HTTPステータス | コード | 発生条件 |
|---|---|---|
| 400 | `INVALID_ROOM_NAME` | ルーム名が制約を満たさない |
| 400 | `INVALID_PASSWORD` | パスワードが制約を満たさない |
| 403 | `INCORRECT_PASSWORD` | パスワードが一致しない |
| 404 | `ROOM_NOT_FOUND` | 指定した名前のルームが存在しない |
| 404 | `USER_NOT_FOUND` | 指定したユーザーが存在しない |
| 409 | `ROOM_FULL` | 既に別の二人が参加している |



#### ルームから退出する。
#### エンドポイント
```/rooms/{room_id}/membership```

#### メソッド
DELETE

#### エラーコード
共通エラーコードに加え、以下のエラーを返す。
| HTTPステータス | コード | 発生条件 |
|---|---|---|
| 404 | `ROOM_NOT_FOUND` | ルームが存在しない |


## WebSocketによるルーム更新通知

視聴履歴の選択や好みの重みの変更はHTTP APIで送信し、参加者の準備状況と推薦結果はWebSocketで通知する。
以下の接続・通知仕様は設計案とする。

### 接続

| 項目 | 内容 |
|---|---|
| エンドポイント | `/rooms/{room_id}/ws` |
| 用途 | 参加者の準備状況と推薦結果をリアルタイムで通知する |
| ユーザー識別 | WebSocketハンドシェイク時に`X-User-ID`ヘッダーを指定する |
| 接続条件 | 接続するユーザーがルームの参加者であること |
| 通信形式 | JSONのテキストメッセージ |

接続成功時はHTTP `101 Switching Protocols`を返す。接続前の認証・権限エラーは、共通エラー仕様のJSONとHTTPステータスで返す。

| HTTPステータス | エラーコード | 条件 |
|---|---|---|
| `404` | `ROOM_NOT_FOUND` | ルームが存在しない、または非参加者 |

接続後の通知にはHTTPステータスを使用せず、`type`でイベントの種類、`data`で内容を表す。この設計では、クライアントからの業務メッセージはWebSocketで送信しない。

### サーバーからクライアントへのイベント

#### 準備状況の更新：`room.updated`

接続直後、および参加者が推薦に使う番組を送信したときに、現在のルーム状態を通知する。状態の変更時は、ルームの接続中の参加者全員に送信する。

```json
{
  "type": "room.updated",
  "data": {
    "room_id": room_123,
    "members": [
      { "user_id": user_1, "ready": true },
      { "user_id": user_2, "ready": false }
    ],
    "status": "waiting_for_partner"
  }
}
```

- `ready`: 推薦に使う番組の選択を送信済みかどうか。
- `status`: `waiting_for_partner`（相手の送信待ち）、`processing`（推薦計算中）、`completed`（推薦完了）、`failed`（推薦失敗）。
- 推薦計算の開始・完了・失敗時にも、更新後の状態を通知する。

#### 推薦完了：`recommendation.completed`

二人分の選択がそろい、推薦の計算が完了したときに、ルームの接続中の参加者全員に通知する。好みの重みの変更による再計算が完了したときも、同じイベントを送信する。

```json
{
  "type": "recommendation.completed",
  "data": {
    "room_id": room_123,
    "items": [
      {
        "series_id": 12,
        "series_title": "番組A"
      }
    ]
  }
}
```

`items`は推薦順に最大10件返す。

#### 推薦失敗：`recommendation.failed`

推薦の計算に失敗したときに、ルームの接続中の参加者全員に通知する。エラーの`code`と`message`は共通エラー仕様と同じ役割とする。

```json
{
  "type": "recommendation.failed",
  "data": {
    "room_id": room_123,
    "error": {
      "code": "INTERNAL_ERROR",
      "message": "推薦の計算に失敗しました。"
    }
  }
}
```

### 接続・再接続時の動作

- 接続直後に、接続したクライアントへ現在のルーム状態を`room.updated`で送信する。
- 再接続時にも`X-User-ID`と参加資格を確認し、現在のルーム状態を送信する。切断中に発生したイベントは再送しない。
- 現在の状態が`completed`の場合は、最新の推薦結果も`recommendation.completed`で送信する。
- 現在の状態が`failed`の場合は、最新の失敗情報も`recommendation.failed`で送信する。
- WebSocketの切断だけではルームから退出した扱いにしない。

## DB design
- user
- episode
- series
新規
- viewing-histories
- rooms
- room-members




