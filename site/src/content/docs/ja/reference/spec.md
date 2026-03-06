---
title: 仕様書
description: sanmon 三門検証スタックの形式仕様
---

## 1. 設計思想

- LLM の内部（重み、アテンション）は形式化できない。
- 代わりに**出力面**を形式化する。出力を構造化し、制約を課し、制約に関する性質を証明する。
- 制約付きデコーディングが、確率的な世界と決定的な世界をつなぐ。
- **定義の一元化**: CUE でスキーマとポリシーの両方を定義する。JSON Schema やバリデーションロジックなど、他の表現はすべて CUE から導出する。

## 2. アーキテクチャ

### 2.1 三つの門

3 つの門はそれぞれ異なるタイミングで動作しますが、CUE という共通の定義元を共有しています。

| 門 | タイミング | 役割 | 技術 |
|---|---|---|---|
| **第一門（構造）** | 生成時 | LLM の出力を JSON Schema に適合させる | CUE → JSON Schema → 制約付きデコーディング |
| **第二門（ポリシー）** | ランタイム | ビジネスルール・安全性ポリシーへの適合を検証 | CUE ランタイム評価（Go） |
| **第三門（証明）** | CI 時 | ポリシー体系のメタ特性を証明 | Lean 4 |

### 2.2 ランタイムパス（リクエスト単位、低レイテンシが求められる）

```
LLM プロバイダー (Bedrock / OpenAI / セルフホスト)
  │ 制約付きデコーディング (CUE 由来の JSON Schema)
  ▼
構造化されたアクション (JSON)
  │
  ▼
sanmon-core (Go ライブラリ、インプロセス)
  │ CUE によるランタイム検証
  │
  ├── OK → アクションを実行
  └── NG → 違反理由を添えて LLM に再生成を要求（最大 N 回）
```

### 2.3 オフラインパス（CI/CD、正確性が求められる）

```
CUE ポリシーファイル (*.cue)
  │
  ├─► cue export --out jsonschema → JSON Schema（第一門の成果物）
  │
  └─► Lean 4 メタ証明
        → ポリシー合成の一貫性（証明済み）
        → 門の単調性（証明済み）
        → 全アクション型へのポリシー定義（証明済み）
```

## 3. 第一門: 制約付きデコーディング

### 目的
トークン生成時に JSON Schema への適合を強制する。事後の検証ではなく、サンプリング分布そのものを変えることで無効なトークンの生成を不可能にする。

### 定義元
JSON Schema は CUE から自動導出される。個別に管理する必要はない。

```
policy/**/*.cue  →  cue export --out openapi  →  JSON Schema
```

### 対応技術
- AWS Bedrock Structured Outputs
- Outlines（OSS、vLLM 連携）
- XGrammar（OSS、文法ベース）

### 保証できること
- JSON Schema への 100% 適合
- 型の正しさ、必須フィールドの存在、enum 値の妥当性
- 幻覚によるフィールド名や無効な構造の排除

### 保証できないこと
- 意味的な正しさ（構造は正しいが値が不適切）
- ビジネスルールの遵守
- 安全性ポリシーの遵守

## 4. 第二門: CUE バリデータ

### 目的
構造的に正しいアクションの意味的な内容を、設定可能なポリシーに照らして検証する。

### 統一アクション形式

AI エージェントのアクションはすべて以下の形式で表現されます。

```json
{
  "action_type": "<ドメイン固有の enum>",
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

### CUE による一元管理

CUE は構造スキーマとセマンティックなポリシーを同じ場所で定義します。

```cue
// 構造（JSON Schema 生成のもと）
#Action: {
    action_type: #BrowserActionType | #ApiActionType | ...
    target:      string
    parameters:  {...}
    context:     #Context
    metadata:    #Metadata
}

// ポリシー（ランタイムで適用）
#BrowserPolicy: {
    url_whitelist: [...string]
    forbidden_selectors: [...string]
    max_input_length: int | *1000
}
```

### ポリシーのディレクトリ構成

```
policy/
├── base/action.cue          # 全ドメイン共通のスキーマ
└── domains/
    ├── browser/policy.cue    # URL ホワイトリスト、禁止セレクタ、入力長制限
    ├── api/policy.cue        # エンドポイント制限、メソッド制限
    ├── database/policy.cue   # 読み取り専用テーブル、WHERE 必須、DROP 禁止
    └── iac/policy.cue        # リソース制限、destroy 禁止、必須タグ
```

### 合成ルール
- **AND**: 該当するポリシーをすべて満たす必要がある
- **継承**: 基本ポリシーにドメイン固有のオーバーライドを重ねる
- **モジュール性**: ドメインを追加しても既存のポリシーに影響しない

### 検証結果

```json
{
  "pass": false,
  "violations": [
    {
      "rule": "browser.url_whitelist",
      "message": "URL 'https://evil.com' は許可パターンに一致しません",
      "path": "parameters.url",
      "severity": "error"
    }
  ]
}
```

### パフォーマンス目標
- CUE 検証のレイテンシ: アクションあたり 10ms 以下

## 5. 第三門: Lean 証明器

### 目的
個々のルールの正しさではなく、ポリシー**体系全体のメタ特性**を証明する。

### Lean で証明する内容

| 特性 | 内容 |
|---|---|
| **一貫性** | 同一ドメイン内のルールが互いに矛盾しない |
| **門の単調性** | 第二門（CUE）を通過するアクションは必ず第一門（JSON Schema）も通過する |
| **完全性** | すべてのアクション型に対して少なくとも 1 つのポリシーが定義されている |
| **合成の安全性** | 基本ポリシーとドメインポリシーをマージしても不変量が保たれる |

### Lean で証明しないこと
- 個別ルールの正しさ（例:「この URL はホワイトリストに含まれているか」）— CUE がランタイムで処理
- LLM の振る舞い — 形式モデルの範囲外

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

-- 中核定理: 安全なアクションは安全な状態を保存する
theorem safe_step_preserves_safety
    (s : State) (a : Action)
    (hs : SafeState s) (ha : SafeAction s a) :
    SafeState (step s a)
```

### 実行方式
- Lean の証明は CI でのみ実行（ランタイムでは実行しない）
- ポリシー変更のたびに PR ゲートとして証明をチェック
- 変更のないポリシーの証明成果物はキャッシュ

## 6. ランタイム: sanmon-core（Go ライブラリ）

### アーキテクチャ

```
sanmon-core (Go ライブラリ)               ← インプロセス、10ms 以下
  ├── CUE ローダー + ポリシー合成
  ├── バリデータ（CUE ランタイム評価）
  ├── JSON Schema エクスポーター
  └── 違反レポーター（構造化出力）

sanmon-server (薄い gRPC ラッパー)        ← 他言語・リモート向け
  └── sanmon-core を利用

sanmon-sdk (将来)                         ← 言語別クライアント
  └── gRPC クライアントラッパー
```

### ライブラリ API（Go）

```go
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

### リトライの仕組み

検証に失敗した場合:
1. 違反理由を収集
2. 元の指示 + 違反内容を構造化した再プロンプトを作成
3. 制約付きデコーディング付きで LLM に再送信
4. 再度検証
5. 最大 N 回まで繰り返す（デフォルト: 3 回）
6. すべて失敗した場合は呼び出し元にエラーを返す

### 組み込みパターン

```
エージェントフレームワーク（任意）
  → sanmon-core.Validate(action)          # インプロセス（推奨）
  → または gRPC 経由で sanmon-server に接続  # リモート
    → OK: エージェントがアクションを実行
    → NG: エージェントが LLM に再生成を要求
```

## 7. ドメイン固有のポリシー

### 7.1 ブラウザ（Playwright / Browser Use）

| ルール | 内容 |
|---|---|
| URL ホワイトリスト | 許可された URL パターン（glob / 正規表現）のみ通過 |
| 禁止セレクタ | クリックや入力を禁止する CSS セレクタ |
| 入力長の上限 | fill 操作で入力できる最大文字数 |
| 危険なスキームの遮断 | `javascript:` や `data:` URI をブロック |
| ページ遷移グラフ | 許可されたナビゲーション経路（将来対応） |

### 7.2 API（MCP / 関数呼び出し）

| ルール | 内容 |
|---|---|
| エンドポイント制限 | 許可されたエンドポイントのみ呼び出し可能 |
| メソッド制限 | エンドポイントごとに使用可能な HTTP メソッドを限定 |
| 認証の必須化 | 変更系の操作には Authorization ヘッダーが必要 |
| ボディスキーマ | リクエストボディが所定のスキーマに一致すること |
| レート制限 | 一定時間内の最大呼び出し回数（将来対応） |

### 7.3 データベース

| ルール | 内容 |
|---|---|
| 読み取り専用テーブル | 指定されたテーブルへの書き込みを禁止 |
| WHERE 句の必須化 | UPDATE / DELETE には WHERE 句が必要 |
| DROP の禁止 | DROP TABLE はデフォルトで無効 |
| 機密カラムの保護 | PII やシークレットを含むカラムへのアクセスを制御 |
| JOIN の深さ制限 | ネストした JOIN の最大数（デフォルト: 3） |

### 7.4 IaC（Terraform / Pulumi）

| ルール | 内容 |
|---|---|
| リソース制限 | 許可されたリソースタイプのみ作成・変更可能 |
| destroy の禁止 | destroy アクションはデフォルトでブロック |
| オープンイングレスの遮断 | 0.0.0.0/0 のセキュリティグループルールを防止 |
| 必須タグ | すべてのリソースに owner・environment・project タグが必要 |
| plan は常に許可 | plan は読み取り専用のため常に許可 |

## 8. 他プロジェクトとの違い

| プロジェクト | アプローチ | sanmon との違い |
|---|---|---|
| Invariant Labs (Snyk) | Python DSL によるルールベース | 形式証明なし。ランタイムチェックのみ |
| AWS Bedrock Automated Reasoning | 事実内容に対する形式論理 | コンテンツの正しさが対象。アクション制約ではない |
| Guardrails AI | バリデータの集合 | 確率的な検出。数学的保証なし |
| AWS Cedar + Lean | 認可ポリシーの検証 | 静的なポリシー向け。AI エージェントのランタイム制約ではない |
| スマートコントラクト検証 | Coq/Lean/Isabelle でコードを検証 | コード自体が対象。AI が生成するアクションではない |

sanmon は、CUE による一元定義 + 制約付きデコーディング + ランタイム検証 + Lean によるメタ証明を組み合わせた唯一のアプローチです。
