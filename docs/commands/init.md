---
codd:
  node_id: "docs:cmd-init"
  title: "init コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---

# teraflow init

## 概要

teraflow プロジェクトを初期化し、設定・状態管理に必要なファイルを生成します。

## 使用方法

```bash
teraflow init [--name <project-name>] [--stage <stage>] [--non-interactive]
```

## フラグ・オプション

- `--name <string>`: プロジェクト名
- `--stage <string>`: 開始ステージ（既定値: `initial_development`）
- `--non-interactive`: 対話入力を省略

## 出力例

```text
Initialized teraflow project files:
  - .github/teraflow.yml
  - .github/project-state.yml
  - docs/shared/01_requirements/index.md
  - docs/guide/golden-principles.md
```

## エラーコード

- `E1001`: 既に初期化済み
- `E0002`: 設定ファイルが見つからない
- `E0003`: 設定ファイル不正

## 使用例

```bash
teraflow init --name "my-project" --non-interactive
teraflow init --name "my-project" --stage continuous_improvement
```
