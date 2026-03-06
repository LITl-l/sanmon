---
title: CLI・API リファレンス
description: sanmon CLI と HTTP サーバー API のリファレンス
---

## CLI: `sanmon`

`sanmon` CLI では、アクションの検証、スキーマのエクスポート、ポリシーの確認ができます。

### `sanmon validate`

アクションを 1 件ずつ、またはディレクトリ単位で検証します。

```bash
# 単一ファイルを検証
sanmon validate --file testdata/valid/browser-navigate.json

# ディレクトリ内の全ファイルを検証
sanmon validate --dir testdata/valid/
```

**出力**: アクションごとの合否と、違反がある場合はその詳細。

### `sanmon schema`

指定したドメインの JSON Schema をエクスポートします。

```bash
# ブラウザドメインのスキーマ
sanmon schema --domain browser

# 他のドメイン
sanmon schema --domain api
sanmon schema --domain database
sanmon schema --domain iac
```

**出力**: JSON Schema が標準出力に出力されます。ファイルに保存するにはリダイレクトしてください。

### `sanmon policy`

現在ロードされているポリシーの設定内容を表示します。

```bash
sanmon policy
```

**出力**: 全ドメインのポリシーとルールの一覧。

---

## HTTP サーバー: `sanmon-server`

sanmon-core を HTTP 経由で利用するためのバリデーションサーバーです。

### 起動方法

```bash
sanmon-server --addr :8080 --policy policy/default-policy.json
```

Just 経由でも起動できます。

```bash
just serve
```

### `POST /validate`

ロード済みのポリシーに対してアクションを検証します。

**リクエスト例**:

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

**レスポンス（合格）**:

```json
{
  "valid": true,
  "violations": []
}
```

**レスポンス（違反あり）**:

```json
{
  "valid": false,
  "violations": [
    {
      "rule": "browser.url_whitelist",
      "message": "URL が許可パターンに一致しません",
      "severity": "error"
    }
  ]
}
```

---

## Go ライブラリ: `sanmon-core`

インプロセスで検証を行うことで、最小のレイテンシを実現します。

```go
import "github.com/LITl-l/sanmon/middleware/pkg/sanmon"
```

### Engine インターフェース

```go
type Engine interface {
    // アクション（JSON バイト列）をポリシーに照らして検証
    Validate(ctx context.Context, action []byte) (*Result, error)

    // ポリシーをディスクから再読み込み
    ReloadPolicies(ctx context.Context) error

    // 指定ドメインの JSON Schema をエクスポート
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

メッセージの詳細は `middleware/proto/guardrails.proto` を参照してください。
