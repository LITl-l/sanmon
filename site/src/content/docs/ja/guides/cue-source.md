---
title: CUE：単一の真実の源
description: CUEがスキーマとポリシー定義をどのように統合するか
---

CUEはsanmonにおける構造とポリシーの両方の単一の真実の源です。他のすべての表現 — JSON Schema、検証ロジック、ドキュメント — はCUE定義から導出されます。

## なぜCUEか？

CUEは設定と検証のために設計されました。型、値、制約を単一の言語で組み合わせます：

- **型と値の統合** — 型は制約であり、値は型である
- **デフォルトでクローズド** — 不明なフィールドは拒否される
- **ハーメチック** — CUEユニバース外からのインポートなし、副作用なし
- **ネイティブGoサポート** — ランタイム評価のためのファーストクラスGoライブラリ

## スキーマ定義

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

## ポリシー構成

```
policy/
├── base/action.cue          # 基本スキーマ（全ドメイン）
└── domains/
    ├── browser/policy.cue    # URLホワイトリスト、禁止セレクタ、入力制限
    ├── api/policy.cue        # エンドポイントホワイトリスト、メソッド制限
    ├── database/policy.cue   # 読み取り専用テーブル、WHERE必須、DROP禁止
    └── iac/policy.cue        # リソースホワイトリスト、destroy禁止、タグ
```

## 導出成果物

単一のCUEファイルセットからsanmonは以下を導出します：

| 成果物 | コマンド | 目的 |
|---|---|---|
| JSON Schema | `just schema` | 制約付きデコーディング（第一門） |
| Go検証 | sanmon-coreライブラリ | ランタイム検証（第二門） |
| Leanモデル | 手動変換（将来：自動化） | メタ証明（第三門） |

## 利点

1. **ドリフトなし** — 構造とポリシーは一緒に定義されるため乖離できない
2. **合成可能性** — CUEの束ベースの型システムが安全なポリシーマージをサポート
3. **ツーリング** — `cue vet`、`cue export`、`cue fmt`が組み込みの検証とフォーマットを提供
4. **パフォーマンス** — CUE評価はランタイム使用に十分高速（目標10ms未満）
