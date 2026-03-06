---
title: ドメインポリシー
description: ブラウザ、API、データベース、IaCドメインの安全性ポリシー
---

sanmonは4つのエージェントドメインのポリシーを提供します。各ドメインは危険なアクションを防止する固有の制約を定義します。

## ブラウザ（Playwright / Browser Use）

| ルール | 説明 |
|---|---|
| URLホワイトリスト | 許可されたURLパターン（glob/正規表現）のみ |
| 禁止セレクタ | クリック/入力が禁止されたCSSセレクタ |
| 入力長制限 | fill操作の最大文字数 |
| 危険スキームブロック | `javascript:`、`data:` URIを禁止 |
| ページ遷移グラフ | 許可されたナビゲーションシーケンス（将来） |

### 例：有効なブラウザアクション

```json
{
  "action_type": "navigate",
  "target": "https://example.com/page",
  "parameters": { "url": "https://example.com/page" },
  "context": { "domain": "browser", "authenticated": true, "session_id": "s1" },
  "metadata": { "timestamp": "2026-02-26T12:00:00Z", "agent_id": "a1", "request_id": "r1" }
}
```

### 例：ブロックされるブラウザアクション

```json
{
  "action_type": "navigate",
  "target": "https://evil.com/phishing",
  "parameters": { "url": "https://evil.com/phishing" }
}
```

違反：`URL 'https://evil.com/phishing' は許可パターンに含まれていません`

## API（MCP / 関数呼び出し）

| ルール | 説明 |
|---|---|
| エンドポイントホワイトリスト | リストされたエンドポイントのみ許可 |
| メソッド制限 | エンドポイントごとのHTTPメソッド制限 |
| 認証要件 | ミューテーションにはAuthorizationヘッダーが必要 |
| ボディスキーマ | リクエストボディが期待されるスキーマに一致する必要がある |
| レートポリシー | 時間ウィンドウあたりの最大呼び出し数（将来） |

## データベース（SQLエージェント）

| ルール | 説明 |
|---|---|
| 読み取り専用テーブル | リストされたテーブルは変更不可 |
| WHERE必須 | UPDATE/DELETEにはWHERE句が必要 |
| DROP禁止 | DROP TABLEはデフォルトで無効 |
| 機密カラム | PII/シークレットカラムのアクセス制御 |
| JOIN深度制限 | ネストJOINの最大数（デフォルト3） |

## IaC（Terraform / Pulumi）

| ルール | 説明 |
|---|---|
| リソースホワイトリスト | リストされたリソースタイプのみ作成/変更可能 |
| destroy禁止 | `destroy`アクションはデフォルトでブロック |
| オープンイングレスブロック | `0.0.0.0/0`セキュリティグループルールを防止 |
| 必須タグ | すべてのリソースに`owner`、`environment`、`project`が必要 |
| planは常に許可 | `plan`は安全な読み取り専用操作 |

## 新しいドメインの追加

新しいドメインを追加するには：

1. `policy/domains/<名前>/policy.cue`にドメイン固有の制約を作成
2. アクションタイプのenumと検証ルールを追加
3. `testdata/valid/`と`testdata/invalid/`にゴールデンテストケースを追加
4. `just test`を実行して検証
5. 形式証明が新しいドメインをカバーする場合、Leanモデルを更新
