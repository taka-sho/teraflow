---
codd:
  node_id: "docs:commands-index"
  title: "teraflow コマンド一覧インデックス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---

# teraflow コマンドリファレンス

## コマンド一覧

| コマンド | 概要 | 主なサブコマンド | リファレンス |
|---------|------|-----------------|-------------|
| `teraflow init` | プロジェクト初期化 | — | [init.md](init.md) |
| `teraflow status` | 現在のステージ・フェーズ表示 | — | [status.md](status.md) |
| `teraflow stage` | ステージ管理 | list, status, advance | [stage.md](stage.md) |
| `teraflow phase` | フェーズ管理 | list, start, complete | [phase.md](phase.md) |
| `teraflow scan` | プロジェクト状態スキャン | — | [scan.md](scan.md) |
| `teraflow rework` | 手戻り管理 | create, list | [rework.md](rework.md) |
| `teraflow config` | 設定管理 | show, set | [config.md](config.md) |
| `teraflow incident` | 障害管理 | create, list, close | [incident.md](incident.md) |
| `teraflow changelog` | 変更ログ管理 | add, generate | [changelog.md](changelog.md) |
| `teraflow label` | GitHubラベル同期 | list, sync | [label.md](label.md) |
| `teraflow discussion` | GitHub Discussion管理 | list, summarize | [discussion.md](discussion.md) |
| `teraflow schedule` | スケジュール管理 | show, update | [schedule.md](schedule.md) |
| `teraflow dashboard` | プロジェクトダッシュボード | show | [dashboard.md](dashboard.md) |
| `teraflow doctor` | 環境・設定健全性チェック | — | [doctor.md](doctor.md) |

## グローバルフラグ

全コマンドで使用できる共通フラグ:

| フラグ | 説明 | デフォルト |
|--------|------|-----------|
| `--config` | 設定ファイルパス | `.github/teraflow.yml` |
| `--format` | 出力形式 (`text` \| `json`) | `text` |
| `--verbose`, `-v` | 詳細出力 | false |

## カテゴリ別コマンド

### プロジェクト初期化・状態確認
- [`teraflow init`](init.md) — 新規プロジェクトを初期化
- [`teraflow status`](status.md) — 現在のステージ・フェーズを確認
- [`teraflow scan`](scan.md) — プロジェクト構成をスキャン
- [`teraflow doctor`](doctor.md) — 環境・設定・整合性を診断

### ライフサイクル管理
- [`teraflow stage`](stage.md) — ステージ遷移管理（initial_development → release → ...）
- [`teraflow phase`](phase.md) — フェーズ進行管理（requirements → implementation → ...）
- [`teraflow schedule`](schedule.md) — マスタースケジュール表示・更新

### 品質・変更管理
- [`teraflow rework`](rework.md) — 手戻り記録・追跡
- [`teraflow incident`](incident.md) — 障害記録・管理（運用ステージ）
- [`teraflow changelog`](changelog.md) — 変更ログ追記・リリースノート生成

### GitHub連携
- [`teraflow label`](label.md) — GitHubラベルを定義に基づき同期
- [`teraflow discussion`](discussion.md) — GitHub Discussionの一覧・要約

### 設定・可視化
- [`teraflow config`](config.md) — teraflow.ymlの設定値を表示・変更
- [`teraflow dashboard`](dashboard.md) — プロジェクト全体の進捗ダッシュボード
