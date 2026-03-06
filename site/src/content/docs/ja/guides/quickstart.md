---
title: クイックスタート
description: sanmonを数分でセットアップして実行する
---

## 前提条件

- [Nix](https://nixos.org/download/)（flakes有効）
- または手動インストール：Go 1.22+、CUE CLI、Lean 4（elan）、Just

## セットアップ

```bash
git clone https://github.com/LITl-l/sanmon.git
cd sanmon
direnv allow   # または: nix develop
```

## ツールチェーンの確認

```bash
# CUEポリシーの検証
just policy-check

# CUEからJSON Schemaを生成
just schema

# gRPC Goコードを生成
just proto

# Lean証明をビルド
just lean-build

# ゴールデンテストスイートを実行
just test
```

## デモの実行

```bash
just demo
```

三門検証のフルデモを実行します：

1. すべての**有効な**テストアクションを検証（すべてパスすることを確認）
2. すべての**無効な**テストアクションを検証（違反の詳細とともにすべて失敗することを確認）
3. ブラウザドメインのJSON Schemaをエクスポート
4. 現在のロード済みポリシーを表示

## HTTPサーバーの起動

```bash
just serve
```

`:8080`でバリデーションサーバーが起動します。アクションを送信して検証：

```bash
curl -X POST http://localhost:8080/validate \
  -H "Content-Type: application/json" \
  -d @testdata/valid/browser-navigate.json
```

## プロジェクト構成

```
sanmon/
├── policy/            # CUE：単一の真実の源（スキーマ + ポリシー）
│   ├── base/              # 基本アクションスキーマ（全ドメイン）
│   └── domains/           # ドメイン固有のポリシー
├── testdata/          # ゴールデンテストスイート（ドメインごとの有効/無効）
├── middleware/         # Go：sanmon-coreライブラリ + gRPCサーバー
│   ├── pkg/sanmon/        # コアバリデーションライブラリ（インプロセス）
│   ├── cmd/sanmon/        # CLIツール
│   └── cmd/server/        # HTTPバリデーションサーバー
├── prover/            # Lean 4：メタ証明
├── schema/generated/  # 導出JSON Schema（Go CLIから）
├── site/              # ドキュメントサイト（Astro Starlight）
└── docs/              # 仕様とアーキテクチャ
```

## ビルドコマンド

| コマンド | 説明 |
|---|---|
| `just build` | CLIとHTTPサーバーをビルド |
| `just test` | ゴールデンテストスイートを実行 |
| `just demo` | エンドツーエンドの検証デモ |
| `just serve` | HTTPバリデーションサーバーを:8080で起動 |
| `just policy-check` | CUEポリシーを検証 |
| `just schema` | CUEからJSON Schemaをエクスポート |
| `just proto` | gRPC Goコードを生成 |
| `just lean-build` | Lean 4の証明をビルド |
| `just clean` | ビルド成果物を削除 |
