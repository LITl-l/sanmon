---
title: CLI・APIリファレンス
description: sanmon CLIツールとHTTPサーバーAPIのリファレンス
---

## CLI：`sanmon`

`sanmon` CLIはアクションの検証、スキーマのエクスポート、ポリシーの確認を行います。

### `sanmon validate`

単一のアクションまたはアクションのディレクトリを検証します。

```bash
# 単一ファイルを検証
sanmon validate --file testdata/valid/browser-navigate.json

# ディレクトリ内のすべてのファイルを検証
sanmon validate --dir testdata/valid/
```

**出力**：各アクションのパス/失敗ステータスと違反の詳細。

### `sanmon schema`

特定のドメインのJSON Schemaをエクスポートします。

```bash
# ブラウザドメインのスキーマをエクスポート
sanmon schema --domain browser

# すべてのドメインのスキーマをエクスポート
sanmon schema --domain api
sanmon schema --domain database
sanmon schema --domain iac
```

**出力**：JSON Schemaを標準出力に出力（必要に応じてファイルにパイプ）。

### `sanmon policy`

現在ロードされているポリシー設定を表示します。

```bash
sanmon policy
```

**出力**：すべてのロード済みドメインポリシーとそのルールのサマリー。

---

## HTTPサーバー：`sanmon-server`

HTTPバリデーションサーバーはsanmon-coreをHTTP経由で公開します。

### サーバーの起動

```bash
sanmon-server --addr :8080 --policy policy/default-policy.json
```

またはJust経由：

```bash
just serve
```

### `POST /validate`

ロード済みポリシーに対してアクションを検証します。

**リクエスト**：

```bash
curl -X POST http://localhost:8080/validate \
  -H "Content-Type: application/json" \
  -d '{
    "action_type": "navigate",
    "target": "https://example.com",
    "parameters": {"url": "https://example.com"},
    "context": {"domain": "browser", "authenticated": true, "session_id": "s1"},
    "metadata": {"timestamp": "2026-02-26T12:00:00Z", "agent_id": "a1", "request_id": "r1"}
  }'
```

**レスポンス（パス）**：

```json
{
  "valid": true,
  "violations": []
}
```

**レスポンス（失敗）**：

```json
{
  "valid": false,
  "violations": [
    {
      "rule": "browser.url_whitelist",
      "message": "URLが許可パターンに含まれていません",
      "severity": "error"
    }
  ]
}
```

---

## Goライブラリ：`sanmon-core`

インプロセス検証（最低レイテンシ）のためにライブラリをインポートします。

```go
import "github.com/LITl-l/sanmon/middleware/pkg/sanmon"
```

### Engineインターフェース

```go
type Engine interface {
    // ロード済みポリシーに対してアクション（JSONバイト列）を検証
    Validate(ctx context.Context, action []byte) (*Result, error)

    // ディスクからポリシーをリロード
    ReloadPolicies(ctx context.Context) error

    // ドメインのJSON Schemaをエクスポート
    ExportJSONSchema(domain string) ([]byte, error)
}
```

### 使い方

```go
engine, err := sanmon.NewEngine("policy/")
if err != nil {
    log.Fatal(err)
}

result, err := engine.Validate(ctx, actionJSON)
if err != nil {
    log.Fatal(err)
}

if !result.Valid {
    for _, v := range result.Violations {
        fmt.Printf("違反: %s — %s\n", v.Rule, v.Message)
    }
}
```

---

## gRPC API

```protobuf
service GuardrailsService {
  rpc Validate(ValidateRequest) returns (ValidateResponse);
  rpc ReloadPolicies(ReloadPoliciesRequest) returns (ReloadPoliciesResponse);
}
```

完全なメッセージ定義は`middleware/proto/guardrails.proto`を参照してください。
