---
codd:
  node_id: "docs:concepts"
  title: "teraflow 概念・用語集"
  depends_on:
    - id: "req:teraflow-overview"
      relation: implements
---

# teraflow 概念・用語集

## 1. teraflow とは何か

teraflow は、日本の大規模SIプロジェクトで発生しやすい「要件〜設計〜実装の断絶」と「手戻りコスト可視化の欠如」を解決するための、ライフサイクル管理CLIである。

中核アプローチは CoDD（Coherence-Driven Development）に基づく。成果物の依存関係を frontmatter で明示し、変更時の影響範囲を追跡しながら、要求から運用までを一貫管理する。

## 2. 3層責務モデル

```text
Layer 1: ライフサイクル管理 (teraflow CLI)
Layer 2: GitHub上の成果物管理 (Issues/Discussions/PRs)
Layer 3: ローカル開発作業
```

- Layer 1: ステージ/フェーズの遷移、ゲート判定、手戻り制御を行う。
- Layer 2: Discussion（入口）→ Issue（作業）→ File（確定記録）の流れを運用する。
- Layer 3: 実装・テスト・レビューなど日常開発を実施し、成果をLayer 2/1に反映する。

## 3. 主要概念

### ステージ (Stage)

プロジェクト全体の大区分。`initial_development`、`release`、`operation` など、ライフサイクル上の状態を表す。

### フェーズ (Phase)

ステージ内の開発工程。例: `requirements` → `basic_design` → `detailed_design` → `implementation` → `testing` → `integration_test`。

### ゲート条件 (Gate)

ステージまたはフェーズ遷移の前提条件。成果物数、必須ファイル存在、ゲートIssue状態などで判定する。

### 手戻り (Rework)

完了済みフェーズへの差し戻し。Issueラベルの変更ではなく、`project-state.yml` のフェーズ履歴を正式に戻し、影響範囲を追跡する。

### 成果物確定

Issue上の議論結果をファイルとして確定し、PRマージをもって正式承認する。確定後はIssueをクローズし、ファイルを正本とする。

### グループ

並行開発チームの分割単位。グループごとにフェーズ進行・担当・成果物を管理し、並行開発を運用する。

### CoDD

依存グラフベースで成果物の整合性と影響分析を行う方法論。`codd scan` / `codd impact` と連携し、変更の波及を可視化する。

## 4. 要件定義概要の統合（旧 `docs/00_overview.md`）

旧 `docs/00_overview.md` の主要内容を本ページへ統合し、概念の入口を一本化した。

### 解決したい課題

- 要件・設計・実装の分断で、意思決定の根拠が失われる
- 手戻り時の影響範囲が見えず、再見積もりが遅れる
- 仕様変更が運用ルールに反映されず、現場判断に依存する

### 方針

- Discussion を入口、Issue を作業場、`docs/` 配下のファイルを正本として運用
- ファイル frontmatter で依存関係を明示し、トレーサビリティを維持
- ステージ/フェーズ遷移を CLI による明示操作で管理

### ライフサイクル全体像

- 要件定義: 期待値と受け入れ条件を確定
- 設計: 構造・責務・品質特性を定義
- 実装/テスト: 変更を検証し、成果物として確定
- リリース/運用: 健全性監視、インシデント対応、継続的改善
