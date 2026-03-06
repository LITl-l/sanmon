---
title: CUE による一元管理
description: CUE でスキーマとポリシーを一元的に定義する仕組み
---

sanmon では、構造（スキーマ）とポリシーの両方を CUE で一元管理しています。JSON Schema、バリデーションロジック、ドキュメントなど、他の成果物はすべて CUE の定義から導出されます。

## CUE を選んだ理由

CUE は設定とバリデーションのために設計された言語で、型・値・制約を統一的に扱えます。

- **型と値が同じ概念** — 型は制約であり、値は型でもある
- **デフォルトでクローズド** — 定義にないフィールドは自動的に拒否
- **副作用なし** — 外部からのインポートや副作用がなく、評価が安全
- **Go との親和性** — Go から直接 CUE を評価できるファーストクラスのライブラリ

## スキーマの定義

CUE では、構造スキーマとセマンティックなポリシーを同じファイルで定義できます。

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

## ポリシーのディレクトリ構成

```
policy/
├── base/action.cue          # 全ドメイン共通のスキーマ
└── domains/
    ├── browser/policy.cue    # URL ホワイトリスト、禁止セレクタ、入力長制限
    ├── api/policy.cue        # エンドポイント制限、メソッド制限
    ├── database/policy.cue   # 読み取り専用テーブル、WHERE 必須、DROP 禁止
    └── iac/policy.cue        # リソース制限、destroy 禁止、必須タグ
```

## CUE から導出される成果物

| 成果物 | コマンド | 用途 |
|---|---|---|
| JSON Schema | `just schema` | 制約付きデコーディング（第一門） |
| Go バリデーション | sanmon-core ライブラリ | ランタイム検証（第二門） |
| Lean モデル | 手動で変換（将来は自動化予定） | メタ証明（第三門） |

## 一元管理のメリット

1. **定義のズレが起きない** — 構造とポリシーを同じ場所で定義するため、乖離しようがない
2. **安全に合成できる** — CUE の束（lattice）ベースの型システムにより、ポリシーのマージが安全
3. **ツールが充実** — `cue vet`・`cue export`・`cue fmt` で検証・変換・整形がすぐにできる
4. **十分に速い** — CUE の評価はランタイム用途に耐える速度（目標: 10ms 以下）
