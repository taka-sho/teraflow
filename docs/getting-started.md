---
codd:
  node_id: "docs:getting-started"
  title: "Getting Started ガイド"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---

# Getting Started

## 前提条件

- Go 1.21 以上
- `git`

確認例:

```bash
go version
git --version
```

## インストール

```bash
go install github.com/taka-sho/teraflow@latest
```

インストール後、`$GOPATH/bin`（または `$(go env GOPATH)/bin`）が `PATH` に含まれていることを確認してください。

## プロジェクト初期化チュートリアル

### 1. プロジェクト初期化

```bash
teraflow init --name "my-project" --non-interactive
```

`init` は `.github/teraflow.yml` と `.github/project-state.yml` を含む初期ファイルを生成します。

### 2. 初期状態確認

```bash
teraflow status
```

現在のステージ、フェーズ、進捗率を表示します。

### 3. ステージ確認

```bash
teraflow stage list
```

定義済みステージの一覧と状態を表示します。

### 4. フェーズ確認

```bash
teraflow phase list
```

現在アクティブなフェーズ情報を表示します。

### 5. 設定確認

```bash
teraflow config show
```

現在の設定値を確認します。

## `.github/teraflow.yml` の主要設定項目

- `version`: 設定フォーマットのバージョン
- `project.name`: プロジェクト名
- `project.description`: プロジェクト概要
- `project.repository`: リポジトリURL
- `confirmation.trigger`: 成果物確定の判定トリガー
- `confirmation.req_trigger`: 要求確定のトリガー
- `ai.default_provider`: AI連携時の既定プロバイダ
- `harness.score_threshold`: 品質スコアの閾値
- `harness.auto_issue`: しきい値未達時のIssue自動作成

## よくあるエラーと対処法

### E0001: Not a teraflow project. Run `teraflow init` first.

- 原因: `.github/teraflow.yml` が存在しないディレクトリでコマンドを実行した。
- 対処:

```bash
teraflow init --name "my-project" --non-interactive
```

### E0002: Configuration file not found

- 原因: 設定ファイルパスが誤っている、またはファイルが未生成。
- 対処: `.github/teraflow.yml` の存在を確認し、必要なら `init` を再実行。

### E0003: Configuration file is invalid

- 原因: YAML形式エラー、必須キー不足。
- 対処: インデントとキー名を見直し、`version` / `project` / `confirmation` 等の必須項目を確認。
