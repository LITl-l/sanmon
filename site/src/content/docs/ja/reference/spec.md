---
title: 仕様書
description: sanmon三門検証スタックの形式仕様
---

## 1. コア設計哲学

- LLMの内部構造（重み、アテンション）は形式化できない。
- 代わりに**出力面**を形式化する：構造化し、制約し、制約に関する特性を証明する。
- 制約付きデコーディングは確率的な世界と決定的な世界の橋渡し。
- **単一の真実の源**：CUEが構造とポリシーの両方を定義する。他のすべての表現（JSON Schema、検証ロジック）はCUEから導出される。

## 2. アーキテクチャ

### 2.1 三つの門（三門）

3つの門は異なるタイミングで動作しますが、単一の真実の源（CUE）を共有しています：

| 門 | タイミング | 目的 | 技術 |
|---|---|---|---|
| **第一門（構造）** | 生成時 | LLM出力をJSON Schemaに適合させる | CUE → JSON Schema → 制約付きデコーディング |
| **第二門（ポリシー）** | ランタイム | ビジネスルールと安全性ポリシーに対してアクションを検証 | CUEランタイム検証（Go） |
| **第三門（証明）** | CI時 | ポリシーシステムのメタ特性を証明 | Lean 4 |

### 2.2 ランタイムパス（リクエストごと、レイテンシ重要）

```
LLMプロバイダー (Bedrock / OpenAI / セルフホスト)
  │ 制約付きデコーディング (CUEから導出されたJSON Schema)
  ▼
構造化アクション (JSON)
  │
  ▼
sanmon-core (Goライブラリ、インプロセス)
  │ CUEランタイム検証
  │
  ├── パス → アクションを実行
  └── 失敗 → 違反理由でLLMに再プロンプト（最大N回リトライ）
```

### 2.3 オフラインパス（CI/CD、正確性重要）

```
CUEポリシーファイル (*.cue)
  │
  ├─► cue export --out jsonschema → JSON Schema (第一門の成果物)
  │
  └─► Lean 4メタ証明
        → 「ポリシー合成は一貫している」(証明済み)
        → 「門の単調性が成立する」(証明済み)
        → 「すべてのアクションタイプにポリシーが定義されている」(証明済み)
```

## 3. 第一門：制約付きデコーディング

### 目的
トークン生成時にLLM出力をJSON Schemaに適合させる。これは事後検証ではない — サンプリング分布を変更して、無効なトークンを不可能にする。

### ソース
JSON Schemaは**CUEから導出**され、別途管理されない。CUEが構造とセマンティクスの両方の単一の真実の源。

```
policy/**/*.cue  →  cue export --out openapi  →  JSON Schema
```

### 技術
- AWS Bedrock Structured Outputs
- Outlines（オープンソース、vLLM統合）
- XGrammar（オープンソース、文法ベース）

### 保証
- 100% JSON Schema適合
- 型の正確性、必須フィールド、enum値の正確性
- 幻覚によるフィールド名や無効な構造なし

### 保証しないこと
- 意味的正確性（有効な構造だが誤った値）
- ビジネスルール準拠
- 安全性特性

## 4. 第二門：CUEバリデータ

### 目的
構造的に有効なアクションの意味的内容を、設定可能なポリシーに対して検証する。

### スキーマ：統一アクション形式

すべてのAIエージェントアクションは以下で表現されます：

```json
{
  "action_type": "<ドメイン固有のenum>",
  "target": "<URL | セレクタ | テーブル | リソース>",
  "parameters": { ... },
  "context": {
    "authenticated": true,
    "session_id": "...",
    "domain": "browser"
  },
  "metadata": {
    "timestamp": "2026-02-26T12:00:00Z",
    "agent_id": "...",
    "request_id": "..."
  }
}
```

### CUE：単一の真実の源

CUEは構造スキーマと意味的ポリシーの両方を一か所で定義します：

```cue
// 構造（JSON Schema生成に供給）
#Action: {
    action_type: #BrowserActionType | #ApiActionType | ...
    target:      string
    parameters:  {...}
    context:     #Context
    metadata:    #Metadata
}

// ポリシー（ランタイムで強制）
#BrowserPolicy: {
    url_whitelist: [...string]
    forbidden_selectors: [...string]
    max_input_length: int | *1000
}
```

### ポリシー構成

```
policy/
├── base/action.cue          # 基本スキーマ（全ドメイン）
└── domains/
    ├── browser/policy.cue    # URLホワイトリスト、禁止セレクタ、入力制限
    ├── api/policy.cue        # エンドポイントホワイトリスト、メソッド制限
    ├── database/policy.cue   # 読み取り専用テーブル、WHERE必須、DROP禁止
    └── iac/policy.cue        # リソースホワイトリスト、destroy禁止、タグ
```

### ポリシー合成
- **AND**：適用可能なすべてのポリシーがパスする必要がある
- **継承**：基本ポリシー + ドメイン固有のオーバーライド
- **モジュール性**：新しいドメインポリシーの追加は既存のものに影響しない

### 検証結果

```json
{
  "pass": false,
  "violations": [
    {
      "rule": "browser.url_whitelist",
      "message": "URL 'https://evil.com' は許可パターンに含まれていません",
      "path": "parameters.url",
      "severity": "error"
    }
  ]
}
```

### パフォーマンス目標
- CUE検証レイテンシ：アクションあたり10ms未満

## 5. 第三門：Lean証明器

### 目的
ポリシーシステムの**メタ特性**を証明する。個別のルールの正確性ではない。

### Leanが証明すること

| 特性 | 説明 |
|---|---|
| **ポリシーの一貫性** | ドメインポリシーセット内の2つのルールが矛盾しない |
| **門の単調性** | 第二門（CUE）をパスするアクションは第一門（JSON Schema）もパスする |
| **ポリシーの完全性** | すべてのアクションタイプに少なくとも1つの適用可能なポリシーが定義されている |
| **合成の安全性** | 基本 + ドメインポリシーのマージが不変量を保持する |

### Leanが証明しないこと
- 個別のルールの正確性（例：「このURLはホワイトリストに含まれている」）— CUEがランタイムで処理
- LLMの振る舞い — 形式モデルの完全に外側

### 形式モデル

```lean
-- アクション型を帰納型として定義
inductive ActionType where
  | browser (a : BrowserAction)
  | api     (a : ApiAction)
  | database (a : DatabaseAction)
  | iac     (a : IacAction)

-- 状態遷移
def step (s : State) (a : Action) : State

-- コア定理：安全なアクションは安全な状態を保持する
theorem safe_step_preserves_safety
    (s : State) (a : Action)
    (hs : SafeState s) (ha : SafeAction s a) :
    SafeState (step s a)
```

### 実行
- Lean証明はCIでのみ実行（ランタイムではない）
- ポリシー変更ごとにPRゲートとして証明チェック
- 変更のないポリシーの証明成果物はキャッシュ

## 6. ランタイム：sanmon-core（Goライブラリ）

### アーキテクチャ

```
sanmon-core (Goライブラリ)              ← インプロセス、10ms未満
  ├── CUEローダー + ポリシーコンポジター
  ├── バリデータ（CUEランタイム評価）
  ├── JSON Schemaエクスポーター
  └── 構造化違反レポーター

sanmon-server (薄いgRPCラッパー)        ← クロス言語/リモート用
  └── sanmon-coreをインポート

sanmon-sdk (将来)                       ← 言語固有クライアント
  └── gRPCクライアントラッパー
```

### ライブラリAPI（Go）

```go
// コア検証
type Engine interface {
    Validate(ctx context.Context, action []byte) (*Result, error)
    ReloadPolicies(ctx context.Context) error
    ExportJSONSchema(domain string) ([]byte, error)
}
```

### gRPC API

```protobuf
service GuardrailsService {
  rpc Validate(ValidateRequest) returns (ValidateResponse);
  rpc ReloadPolicies(ReloadPoliciesRequest) returns (ReloadPoliciesResponse);
}
```

### リトライループ

検証失敗時：
1. 違反理由を収集
2. 再プロンプトを構築：元の指示 + 構造化された違反フィードバック
3. 制約付きデコーディングでLLMに再送信
4. 再度検証
5. 最大N回繰り返し（設定可能、デフォルト3）
6. すべてのリトライが失敗した場合、呼び出し元にエラーを返す

### 統合パターン

```
エージェントフレームワーク（任意）
  → sanmon-core.Validate(action)          # インプロセス（推奨）
  → または gRPCクライアント呼び出し         # リモート
    → パス: エージェントがアクションを実行
    → 失敗: エージェントがLLMに再プロンプト
```

## 7. ドメイン固有のポリシー

### 7.1 ブラウザ（Playwright / Browser Use）

| ルール | 説明 |
|---|---|
| URLホワイトリスト | 許可されたURLパターン（glob/正規表現）のみ |
| 禁止セレクタ | クリック/入力が禁止されたCSSセレクタ |
| 入力長制限 | fill操作の最大文字数 |
| 危険スキームブロック | `javascript:`、`data:` URIを禁止 |
| ページ遷移グラフ | 許可されたナビゲーションシーケンス（将来） |

### 7.2 API（MCP / 関数呼び出し）

| ルール | 説明 |
|---|---|
| エンドポイントホワイトリスト | リストされたエンドポイントのみ許可 |
| メソッド制限 | エンドポイントごとのHTTPメソッド制限 |
| 認証要件 | ミューテーションにはAuthorizationヘッダーが必要 |
| ボディスキーマ | リクエストボディが期待されるスキーマに一致する必要がある |
| レートポリシー | 時間ウィンドウあたりの最大呼び出し数（将来） |

### 7.3 データベース

| ルール | 説明 |
|---|---|
| 読み取り専用テーブル | リストされたテーブルは変更不可 |
| WHERE必須 | UPDATE/DELETEにはWHERE句が必要 |
| DROP禁止 | DROP TABLEはデフォルトで無効 |
| 機密カラム | PII/シークレットカラムのアクセス制御 |
| JOIN深度制限 | ネストJOINの最大数（デフォルト3） |

### 7.4 IaC（Terraform / Pulumi）

| ルール | 説明 |
|---|---|
| リソースホワイトリスト | リストされたリソースタイプのみ作成/変更可能 |
| destroy禁止 | destroyアクションはデフォルトでブロック |
| オープンイングレスブロック | 0.0.0.0/0セキュリティグループルールを防止 |
| 必須タグ | すべてのリソースにowner、environment、projectが必要 |
| planは常に許可 | planは安全な読み取り専用操作 |

## 8. 差別化

| プロジェクト | アプローチ | sanmonとの違い |
|---|---|---|
| Invariant Labs (Snyk) | Python DSLルールベース | 形式証明なし、ランタイムチェックのみ |
| AWS Bedrock Automated Reasoning | 事実内容の形式論理 | コンテンツ検証、アクション制約ではない |
| Guardrails AI | バリデータコレクション | 確率的検出、数学的保証なし |
| AWS Cedar + Lean | 認可ポリシー検証 | 静的ポリシー、AIエージェントランタイム制約ではない |
| スマートコントラクト検証 | Coq/Lean/Isabelle | コード自体を検証、AI生成アクションではない |

sanmonはCUE定義 + 制約付きデコーディング + ランタイム検証 + メタレベル形式証明（Lean）を組み合わせた唯一のプロジェクトです。
