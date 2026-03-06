---
title: ドメインポリシー
description: ブラウザ・API・データベース・IaC の各ドメインにおける安全性ポリシー
---

sanmon は 4 つのエージェントドメイン向けのポリシーを標準で用意しています。ドメインごとに、危険なアクションを防ぐための制約が定義されています。

## ブラウザ（Playwright / Browser Use）

| ルール | 内容 |
|---|---|
| URL ホワイトリスト | 許可された URL パターン（glob / 正規表現）のみ通過 |
| 禁止セレクタ | クリックや入力を禁止する CSS セレクタ |
| 入力長の上限 | fill 操作で入力できる最大文字数 |
| 危険なスキームの遮断 | `javascript:` や `data:` URI をブロック |
| ページ遷移グラフ | 許可されたナビゲーション経路（将来対応） |

### 例: 許可されるブラウザアクション

```json
{
  "action_type": "navigate",
  "target": "https://example.com/page",
  "parameters": { "url": "https://example.com/page" },
  "context": { "domain": "browser", "authenticated": true, "session_id": "s1" },
  "metadata": { "timestamp": "2026-02-26T12:00:00Z", "agent_id": "a1", "request_id": "r1" }
}
```

### 例: 拒否されるブラウザアクション

```json
{
  "action_type": "navigate",
  "target": "https://evil.com/phishing",
  "parameters": { "url": "https://evil.com/phishing" }
}
```

違反: `URL 'https://evil.com/phishing' は許可パターンに一致しません`

## API（MCP / 関数呼び出し）

| ルール | 内容 |
|---|---|
| エンドポイント制限 | 許可されたエンドポイントのみ呼び出し可能 |
| メソッド制限 | エンドポイントごとに使用可能な HTTP メソッドを限定 |
| 認証の必須化 | 変更系の操作には Authorization ヘッダーが必要 |
| ボディスキーマ | リクエストボディが所定のスキーマに一致すること |
| レート制限 | 一定時間内の最大呼び出し回数（将来対応） |

## データベース（SQL エージェント）

| ルール | 内容 |
|---|---|
| 読み取り専用テーブル | 指定されたテーブルへの書き込みを禁止 |
| WHERE 句の必須化 | UPDATE / DELETE には WHERE 句が必要 |
| DROP の禁止 | DROP TABLE はデフォルトで無効 |
| 機密カラムの保護 | PII やシークレットを含むカラムへのアクセスを制御 |
| JOIN の深さ制限 | ネストした JOIN の最大数（デフォルト: 3） |

## IaC（Terraform / Pulumi）

| ルール | 内容 |
|---|---|
| リソース制限 | 許可されたリソースタイプのみ作成・変更可能 |
| destroy の禁止 | `destroy` アクションはデフォルトでブロック |
| オープンイングレスの遮断 | `0.0.0.0/0` のセキュリティグループルールを防止 |
| 必須タグ | すべてのリソースに `owner`・`environment`・`project` タグが必要 |
| plan は常に許可 | `plan` は読み取り専用のため常に許可 |

## 新しいドメインを追加するには

1. `policy/domains/<名前>/policy.cue` にドメイン固有の制約を定義
2. アクション型の enum とバリデーションルールを追加
3. `testdata/valid/` と `testdata/invalid/` にテストケースを追加
4. `just test` で動作を確認
5. Lean の形式証明で新ドメインをカバーする場合はモデルを更新
