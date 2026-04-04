## 19. トレーサビリティ（ファイルベース）

3層責務モデル（セクション1）に基づき、トレーサビリティは Issue 間のリンクではなく **ファイル間の CoDD frontmatter** で管理する。

### 19.1 追跡構造

```
docs/01_requirements/REQ-0010.md（要求）
 └── docs/02_requirement-definition/REQDEF-0020.md（要件）
      ├── depends_on: req:REQ-0010
      └── docs/03_basic-design/BD-0030.md（基本設計）
           ├── depends_on: reqdef:REQDEF-0020
           └── docs/04_detailed-design/DD-0040.md（詳細設計）
                ├── depends_on: design:BD-0030
                └── src/ のコード（実装）
```

### 19.2 frontmatter の標準構造

全ての成果物ファイルに以下の frontmatter を付与する。

```yaml
---
codd:
  node_id: "{type}:{file_id}"      # 例: "req:REQ-0010", "design:BD-0030"
  depends_on:
    - id: "{上流ファイルのnode_id}"
      relation: derives_from        # derives_from / implements / traces_to / verifies
source_issue: {Issue番号}            # 元のIssue番号（参考情報）
confirmed_at: "{ISO8601}"           # 確定日時
confirmed_by: "{GitHubユーザー名}"   # 確定者
---
```

### 19.3 `codd scan` による依存グラフ構築

全成果物ファイルの frontmatter を走査し、依存グラフを構築する。手戻り時の `codd impact` はこのグラフを用いて影響範囲を特定する。

継続的改善サイクルの成果物にはサイクルID（Milestone名）を frontmatter に追加し、サイクル単位のトレーサビリティも確保する。

---

## 20. GitHub Actionsワークフロー一覧

| # | ファイル名 | トリガー | ステージ | 機能 |
|---|---|---|---|---|
| 1 | `teraflow-stage-guard.yml` | Issue作成時 | 全ステージ | ステージ不一致Issueの拒否 |
| 2 | `teraflow-phase-guard.yml` | Issue作成時 | 全ステージ | フェーズ不一致Issueの拒否 |
| 3 | `teraflow-req-agent.yml` | Discussion作成/コメント | 初期開発, 継続的改善 | 要求整理AI対話（Discussion上） |
| 4 | `teraflow-req-finalize.yml` | Discussionコメント「要求確定」 | 初期開発, 継続的改善 | REQ-*.md生成+Issue作成+PR |
| 5 | `teraflow-phase-agent.yml` | Issueコメント（bot以外） | 全ステージ | Issue上でのAI対話（要件定義〜テスト） |
| 6 | `teraflow-phase-finalize.yml` | Issueコメント「確定」 | 全ステージ | 成果物ファイル生成+PR+Issueクローズ |
| 7 | `teraflow-index-update.yml` | develop push (docs/変更時) | 全ステージ | 各フェーズのindex.md自動再生成 |
| 8 | `teraflow-implement-agent.yml` | `agent:実装開始`ラベル | 初期開発, 継続的改善 | 実装→PR自動生成 |
| 9 | `teraflow-ci-fix-agent.yml` | `agent:CIを修正`ラベル | 全ステージ | CI失敗自動修正 |
| 10 | `teraflow-review-agent.yml` | PR作成時 | 全ステージ | 自動コードレビュー |
| 11 | `teraflow-review-fix-agent.yml` | `agent:レビュー修正`ラベル | 全ステージ | レビュー指摘自動修正 |
| 12 | `teraflow-conflict-agent.yml` | `agent:コンフリクト解消`ラベル | 全ステージ | コンフリクト自動解消 |
| 13 | `teraflow-stuck-monitor.yml` | 定期実行（15分間隔） | 全ステージ | パイプライン停滞検知 |
| 14 | `teraflow-rework-impact.yml` | `rework`ラベル付きIssue | 全ステージ | CoDD影響分析+記録 |
| 15 | `teraflow-rework-approve.yml` | 手戻りIssueに「手戻り承認」コメント | 全ステージ | フェーズ後退PR作成+履歴更新 |
| 16 | `teraflow-dashboard.yml` | develop push/日次 | 全ステージ | ダッシュボード更新 |
| 17 | `teraflow-gate-check.yml` | gateラベルIssueクローズ | 全ステージ | ゲート条件チェック（ファイルベース） |
| 18 | `teraflow-schedule-update.yml` | フェーズ遷移時 | 全ステージ | スケジュール実績自動記録 |
| 19 | `teraflow-incident-agent.yml` | `incident`ラベル付きIssue | 運用 | 障害調査エージェント |
| 20 | `teraflow-maintenance-agent.yml` | `agent:保守実行`ラベル | 保守 | 保守作業エージェント |
| 21 | `teraflow-scoring.yml` | 週次スケジュール | 保守 | 定期スコアリング+自動Issue起票 |

**#5 `teraflow-phase-agent.yml` の詳細**:

要件定義〜テストの各フェーズで、Issue コメントに応じて AI エージェントが対話する。フェーズごとに異なるシステムプロンプトを使用し、そのフェーズに適した質問・整理を行う。要求フェーズは Discussion 上で対話するため対象外。

**#6 `teraflow-phase-finalize.yml` の詳細**:

全フェーズ共通の確定ワークフロー。「確定」コメントをトリガーに以下を実行する。

1. Issue のフェーズラベルから対象フェーズを判定
2. Issue の全会話履歴を取得
3. AI がフェーズに応じた成果物ファイルを生成（REQ/REQDEF/BD/DD/TEST）
4. CoDD frontmatter を自動付与（`depends_on` で上流成果物にリンク）
5. 適切な `docs/` 配下にファイルを配置
6. PR を作成
7. PR マージ後に Issue を自動クローズ
8. クローズコメントに成果物ファイルへのリンクを記載

---

## 21. 導入ロードマップ

### Phase 1: 基盤構築（初期開発ステージ開始前）

`teraflow init` で以下を整備:
- ディレクトリ構成、設定ファイル群（`project-state.yml` 含む）
- CLAUDE.md + Golden Principles
- Issueテンプレート（初期開発用）、ラベル一括作成
- ステージガード + フェーズガード Actions (#1, #2)
- 要求整理エージェント Actions (#3, #4)
- Issue上AI対話 + 確定フロー Actions (#5, #6, #7)
- ダッシュボード Actions (#16)

### Phase 2: Agent実装の導入（初期開発・実装フェーズ序盤）

- 実装エージェント (#8) を少数Issueで試行
- CI自動修正 (#9) を追加
- 全PRは人間レビュー必須（agent:supervised）

### Phase 3: Agent安定化（初期開発・実装フェーズ中盤）

- 自動レビュー + レビュー指摘修正 (#10, #11)
- コンフリクト解消 (#12)、停滞監視 (#13)
- Agent比率を段階的に引き上げ

### Phase 4: リリース準備（移行・リリースステージ）

- 移行・リリースステージ用テンプレート・ドキュメント整備
- リリースフェーズのゲート条件設定

### Phase 5: 運用体制構築（運用ステージ開始）

- 障害報告テンプレート有効化
- 障害調査エージェント (#19) 有効化
- `incident-log.yml` の運用開始
- ダッシュボードに障害統計セクション追加

### Phase 6: 継続的改善の本格稼働（継続的改善ステージ）

- 改善要求テンプレート有効化
- サイクル管理（`teraflow cycle`）の運用開始
- リリースフェーズの追加（リグレッションテスト含む）
- 過去サイクルの実績に基づく予測精度向上

### Phase 7: 保守体制構築（保守ステージ有効化）

- `teraflow stage activate 保守`
- 保守作業テンプレート有効化
- 定期スコアリング (#21) 有効化
- 保守作業エージェント (#20) 有効化
- スコア閾値による自動Issue起票

---

## 22. Secrets・環境変数

| Secret名 | 用途 | 必須 |
|---|---|---|
| `ANTHROPIC_API_KEY` | AIエージェントのAPI呼び出し | ✅ |
| `SLACK_WEBHOOK_URL` | 通知（設定時のみ） | ❌ |

`GITHUB_TOKEN` はActionsのデフォルトトークンを使用。

---

## 23. 実装上の注意事項

### 23.1 言語非依存の維持

CLIツールの実装言語は任意だが、管理対象プロジェクトの言語には依存しない設計にすること。CIチェック項目の実際のコマンドはプロジェクトごとに `teraflow.yml` で設定する。テンプレート類はTerasolunaの3層構造の概念レベルで記述し、特定言語の構文を含めない。

### 23.2 CoDDとの統合

`teraflow` はCoDD（`codd-dev` パッケージ）を内部依存として利用する。`teraflow init` 時に `codd init` も同時実行する。手戻りIssue作成時に `codd impact` を呼び出す。設計書のfrontmatterはCoDD仕様に準拠する。

### 23.3 誤発火防止

- 「要求確定」トリガーはコメント本文が厳密に一致する場合のみ発火
- `github-actions[bot]` のコメントはトリガー対象外
- 全ワークフローに `concurrency` 設定を適用し重複実行を防止

### 23.4 セキュリティ

- APIキーはRepository Secretsにのみ格納しログに出力しない
- Agentが生成するコードにも人間と同じCIチェックを全て適用
- 自動マージは `autonomous` 信頼度 + CI全パス + 変更行数制限の条件下でのみ許可

### 23.5 ステージ遷移の安全性

- ステージ遷移は必ずPR経由で行い、`project-state.yml` の変更にはレビューを必須とする
- ステージの凍結（freeze）は即時反映だが、解除にはPR+レビューを必要とする
- 廃止ステージへの遷移時、`on_activate.freeze_stages` に指定されたステージが自動凍結される
