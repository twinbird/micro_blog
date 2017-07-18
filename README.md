# すいったー

## 機能

 * ユーザ登録
 * ログイン
 * ログアウト
 * すいーと
 * ユーザ検索
 * フォロー
 * アンフォロー
 * タイムライン

## DB定義

MySQL前提.
全項目NOT NULL.

----------------------

Users

| 項目名          | 型          | 内容                                          | 属性        |
|-----------------|-------------|-----------------------------------------------|-------------|
| id              | SERIAL      | ユーザ固有のID                                | PRImary KEY |
| name            | VARCHAR(30) | 表示ユーザ名                                  | -           |
| email           | VARCHAR(50) | ユーザメールアドレス                          | UNIQUE      |
| hashed_password | VARCHAR(64) | password + saltでSHA1ハッシュされたパスワード | -           |
| salt            | VARCHAR(30) | SHA1ハッシュされたパスワード                  | -           |
| created_at      | DATETIME    | 作成日時                                      | -           |

----------------------

Posts

| 項目名       | 型              | 内容                 | 属性        |
|--------------|-----------------|----------------------|-------------|
| id           | SERIAL          | Post固有のID         | PRIMARY KEY |
| post_user_id | BIGINT UNSIGNED | ポストしたユーザのID | -           |
| messege      | VARCHAR(140)    | メッセージ           | -           |
| created_at   | DATETIME        | 作成日時             | -           |

----------------------

Followers

| 項目名           | 型              | 内容                     | 属性         |
|------------------|-----------------|--------------------------|--------------|
| user_id          | BIGINT UNSIGNED | フォローされるユーザのID | PRIMARY KEY  |
| follower_user_id | BIGINT UNSIGNED | フォローするユーザのID   | PRIMARY KEY  |
| created_at       | DATETIME        | 作成日時                 | -            |
