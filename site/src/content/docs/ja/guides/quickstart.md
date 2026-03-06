---
title: クイックスタート
description: sanmon を数分でセットアップして動かす
---

## 前提条件

- [Nix](https://nixos.org/download/)（flakes を有効化済み）
- Nix を使わない場合: Go 1.22 以上、CUE CLI、Lean 4（elan）、Just を個別にインストール

## セットアップ

```bash
git clone https://github.com/LITl-l/sanmon.git
cd sanmon
direnv allow   # または nix develop
```

## ツールチェーンの確認

```bash
# CUE ポリシーの検証
just policy-check

# JSON Schema の生成
just schema

# gRPC の Go コード生成
just proto

# Lean 証明のビルド
just lean-build

# テストの実行
just test
```

## デモを動かす

```bash
just demo
```

三門検証のデモが一通り実行されます。

1. **有効な**テストアクションをすべて検証（すべてパスすることを確認）
2. **無効な**テストアクションをすべて検証（違反内容とともにすべて失敗することを確認）
3. ブラウザドメインの JSON Schema をエクスポート
4. 現在のポリシー設定を表示

## HTTP サーバーを起動する

```bash
just serve
```

ポート `:8080` でバリデーションサーバーが立ち上がります。

```bash
curl -X POST http://localhost:8080/validate \
  -H "Content-Type: application/json" \
  -d @testdata/valid/browser-navigate.json
```

## プロジェクト構成

```
sanmon/
├── policy/            # CUE: スキーマとポリシーの定義元
│   ├── base/              # 全ドメイン共通のアクションスキーマ
│   └── domains/           # ドメイン固有のポリシー
├── testdata/          # ゴールデンテスト（ドメインごとに有効/無効）
├── middleware/         # Go: sanmon-core ライブラリ + サーバー
│   ├── pkg/sanmon/        # コアバリデーションライブラリ（インプロセス）
│   ├── cmd/sanmon/        # CLI ツール
│   └── cmd/server/        # HTTP バリデーションサーバー
├── prover/            # Lean 4: メタ証明
├── schema/generated/  # Go CLI が出力する JSON Schema
├── site/              # ドキュメントサイト（Astro Starlight）
└── docs/              # 仕様書・設計ドキュメント
```

## ビルドコマンド一覧

| コマンド | 内容 |
|---|---|
| `just build` | CLI と HTTP サーバーをビルド |
| `just test` | テストを実行 |
| `just demo` | 三門検証のデモを実行 |
| `just serve` | HTTP サーバーを :8080 で起動 |
| `just policy-check` | CUE ポリシーを検証 |
| `just schema` | JSON Schema をエクスポート |
| `just proto` | gRPC の Go コードを生成 |
| `just lean-build` | Lean 4 の証明をビルド |
| `just clean` | ビルド成果物を削除 |
