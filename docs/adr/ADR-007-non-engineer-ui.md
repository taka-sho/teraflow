---
codd:
  node_id: "adr:007-non-engineer-ui"
  title: "ADR-007: 非エンジニア向けUI設計方針（Issue/Discussion活用）"
  depends_on:
    - id: "req:phase2-github-integration"
      relation: implements
    - id: "adr:006-event-driven"
      relation: refines
---

# ADR-007: 非エンジニア向けUI設計方針

## ステータス

提案（Proposed）

## コンテキスト

PM/PMOはターミナル操作ができない。Phase2ではGitHub.com標準UIのみでプロジェクト管理の全操作（ステージ遷移・フェーズ管理・手戻り・インシデント等）を行えるようにする必要がある。

## 決定

**Issueテンプレート + ラベル + Discussionカテゴリ + GitHub Projects で全操作をカバーする。**

### UI要素の役割分担

| UI要素 | 用途 | 利点 |
|--------|------|------|
| Issueテンプレート | 構造化された操作入力（フェーズ開始/完了/手戻り等） | フォーム形式で入力ミスを防止 |
| ラベル | 状態表示 + 操作トリガー（W-001ハイブリッド方式） | 視覚的に現在状態を把握 |
| Discussion | 非構造的議論（要件議論/設計議論/振り返り） | AI対話Agent対応 |
| GitHub Projects | カンバンビュー（ステージ×フェーズ） | 全体進捗の俯瞰 |
| GitHub Pages | ダッシュボード表示 | 詳細メトリクス・グラフ |

### Issueテンプレートの設計原則

1. **フォーム形式**: YAML形式のIssueテンプレート（`type: input`, `type: dropdown`）を使用
2. **自動ラベル**: テンプレート選択時にラベルを自動付与し、Actionsトリガーに使用
3. **バリデーション**: 必須フィールドでフォームバリデーション
4. **プレフィル**: グループ名・フェーズ名をドロップダウンで選択式に

```yaml
# 例: フェーズ完了報告テンプレート
name: "フェーズ完了報告"
description: "フェーズの完了を報告します"
labels: ["phase-complete"]
body:
  - type: dropdown
    id: group
    attributes:
      label: "グループ"
      options:
        - group-auth
        - group-api
    validations:
      required: true
  - type: dropdown
    id: phase
    attributes:
      label: "完了フェーズ"
      options:
        - requirements
        - basic_design
        - detailed_design
        - implementation
        - test
    validations:
      required: true
  - type: textarea
    id: summary
    attributes:
      label: "完了サマリ"
```

### 代替案（却下）

| 案 | 却下理由 |
|----|---------|
| 独自Web UI | Phase4の範囲。GitHub.com UIで十分 |
| Slack Bot | GitHub外への依存。SSoT原則に反する |
| GitHub CLI Web | 非エンジニアにCLI操作は不適切 |

## 影響

- Phase2追加分の6テンプレートのメンテナンスが必要
- ラベル体系の厳密管理（ラベル名の変更はActionsトリガーに影響）
- Issueテンプレートのドロップダウン選択肢はgroups.ymlから手動同期（Phase3でApp化後に自動化）
