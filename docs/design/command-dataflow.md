---
codd:
  node_id: "design:command-dataflow"
  title: "コマンド→内部処理のデータフロー"
  depends_on:
    - id: "design:system-overview"
      relation: derives_from
    - id: "req:cli-project-mgmt"
      relation: implements
    - id: "req:schedule-rework-ops"
      relation: implements
---

# コマンド→内部処理のデータフロー

## 概要

teraflow CLIの主要コマンドについて、ユーザー入力から内部処理、ファイル更新、外部システム連携までのデータフローを定義する。Phase1対象機能（cmd_081マトリクスのP1列）を中心に記述する。

## 1. teraflow init

プロジェクト初期化。.teraflow/配下にYAMLファイル群とGitHubリソースを生成する。

```
ユーザー: teraflow init --project-name "my-project"
    │
    ▼
[cmd/init.go]
    │ フラグ解析: --project-name
    ▼
[core/setup.go] SetupExecutor.Init()
    │
    ├──(1) .teraflow/ ディレクトリ作成
    │      ├── teraflow.yml（デフォルト設定生成）
    │      ├── project-state.yml（初期状態: initial_development/requirements_definition）
    │      ├── master-schedule.yml（空テンプレート）
    │      ├── groups.yml（空: グループ未定義）
    │      └── changelog/（空ディレクトリ）
    │
    ├──(2) gh 認証チェック
    │      [github/gh.go] → gh auth status
    │
    ├──(3) GitHub リソース生成（--skip-github で省略可）
    │      ├── [github/label.go] → gh label create × 50+
    │      ├── Issueテンプレート生成 → .github/ISSUE_TEMPLATE/*.yml
    │      └── CODEOWNERS生成 → .github/CODEOWNERS
    │
    └──(4) 変更ログ記録
           [store/changelog_writer.go] → .teraflow/changelog/YYYY-MM.jsonl
           イベント: {"type": "project.init", "project": "my-project", ...}
```

### 出力ファイル

| ファイル | 内容 |
|---------|------|
| `.teraflow/teraflow.yml` | メイン設定（AI provider, gates, confirmation等） |
| `.teraflow/project-state.yml` | プロジェクト状態（stages, lifecycle） |
| `.teraflow/master-schedule.yml` | スケジュール（空テンプレート） |
| `.teraflow/groups.yml` | グループ定義（空） |
| `.teraflow/changelog/YYYY-MM.jsonl` | 変更ログ（init記録） |
| `.github/ISSUE_TEMPLATE/*.yml` | Issueテンプレート 11種 |
| `.github/CODEOWNERS` | コードオーナー設定 |

---

## 2. teraflow stage advance

現在のステージを次のステージへ遷移させる。ゲート条件をチェックする。

```
ユーザー: teraflow stage advance
    │
    ▼
[cmd/stage/advance.go]
    │
    ▼
[core/stage.go] StageManager.Advance()
    │
    ├──(1) 現在ステージ取得
    │      [store/yaml_store.go] → .teraflow/project-state.yml 読み込み
    │      現在: initial_development (active)
    │
    ├──(2) ゲート条件評価
    │      [core/gate.go] GateEvaluator.Evaluate()
    │      ├── teraflow.yml の gates 設定を読み込み
    │      ├── 条件例: all_groups_phase == "integration_test"
    │      ├── [store/yaml_store.go] → groups.yml, project-state.yml 参照
    │      └── 結果: pass / fail（失敗理由付き）
    │
    ├──(3) ゲート通過時: ステージ遷移
    │      ├── project-state.yml 更新
    │      │   stages[current].status = "completed"
    │      │   stages[next].status = "active"
    │      │   lifecycle.current_stage = next
    │      │
    │      └── 変更ログ記録
    │          {"type": "stage.advance", "from": "initial_development", "to": "release"}
    │
    └──(4) ゲート不通過時: エラー出力
           未達条件の一覧を表示
```

---

## 3. teraflow phase advance --group GROUP

指定グループのフェーズを次へ進める。

```
ユーザー: teraflow phase advance --group group-auth
    │
    ▼
[cmd/phase/advance.go]
    │ フラグ解析: --group
    ▼
[core/phase.go] PhaseManager.Advance(groupID)
    │
    ├──(1) グループ・フェーズ状態取得
    │      [store] → project-state.yml, groups.yml
    │      group-auth: 現在フェーズ = external_design
    │
    ├──(2) フェーズ遷移可否チェック
    │      ├── 現在フェーズの成果物確定状況
    │      │   planned_artifacts vs confirmed_artifacts
    │      ├── 同期ポイントチェック（sync_points）
    │      │   group-auth が group-api の external_design 完了を待つ場合
    │      └── ロール権限チェック（role.check）
    │
    ├──(3) 遷移実行
    │      ├── project-state.yml 更新
    │      │   group-auth.current_phase = "implementation"
    │      │   group-auth.phase_history += {phase: "external_design", completed_at: now}
    │      │
    │      ├── master-schedule.yml 更新
    │      │   group-auth.phases.external_design.actual_end = now
    │      │   group-auth.phases.implementation.actual_start = now
    │      │
    │      └── 変更ログ記録
    │          {"type": "phase.advance", "group": "group-auth",
    │           "from": "external_design", "to": "implementation"}
    │
    └──(4) スケジュール再予測（オプション: --predict）
           [core/schedule.go] → [ai/] → 完了予測を再計算
```

---

## 4. teraflow rework create

手戻りIssueを起票し、影響分析を実行する。

```
ユーザー: teraflow rework create --group group-auth --target-phase external_design --reason "API仕様変更"
    │
    ▼
[cmd/rework/create.go]
    │ フラグ解析: --group, --target-phase, --reason
    ▼
[core/rework.go] ReworkManager.Create(groupID, targetPhase, reason)
    │
    ├──(1) 現在状態確認
    │      [store] → project-state.yml
    │      group-auth: 現在 implementation → external_design への手戻り
    │
    ├──(2) 影響分析（CoDD Adapter）
    │      [codd/analyzer.go] Analyzer.Impact("group-auth:external_design")
    │      ├── docs/ 配下のfrontmatter依存グラフを走査
    │      ├── 影響分類:
    │      │   Green: 直接影響（external_design の成果物）
    │      │   Amber: 間接影響（implementation の成果物のうち依存するもの）
    │      │   Gray: 影響なし
    │      └── 結果: ImpactReport
    │
    ├──(3) GitHub Issue 起票
    │      [github/issue.go] → gh issue create
    │      ├── タイトル: "[Rework] group-auth: external_design ← API仕様変更"
    │      ├── ラベル: rework, group:auth, phase:external-design
    │      ├── 本文: 影響分析結果を含む
    │      └── 戻り値: Issue URL, Issue番号
    │
    ├──(4) 状態更新
    │      ├── project-state.yml: 手戻り記録を追加
    │      └── master-schedule.yml: 予測への影響を反映
    │
    └──(5) 変更ログ記録
           {"type": "rework.create", "group": "group-auth",
            "target_phase": "external_design", "issue_number": 42,
            "impact": {"green": 3, "amber": 5, "gray": 12}}
```

---

## 5. teraflow rework approve

手戻りを承認し、フェーズを後退させるPRを作成する。

```
ユーザー: teraflow rework approve --issue 42
    │
    ▼
[cmd/rework/approve.go]
    │
    ▼
[core/rework.go] ReworkManager.Approve(issueNumber)
    │
    ├──(1) 手戻りIssue情報取得
    │      [github/issue.go] → gh issue view 42 --json
    │
    ├──(2) フェーズ後退実行
    │      project-state.yml 更新:
    │      group-auth.current_phase = "external_design" (後退)
    │      group-auth.phase_history += {phase: "implementation", reverted_at: now, reason: "rework#42"}
    │
    ├──(3) PR作成（フェーズ状態変更のコミット）
    │      ├── git checkout -b rework/42-external-design
    │      ├── git add .teraflow/project-state.yml
    │      ├── git commit -m "rework(#42): revert group-auth to external_design"
    │      └── [github/pr.go] → gh pr create
    │
    └──(4) 変更ログ記録
           {"type": "rework.approve", "issue_number": 42,
            "group": "group-auth", "reverted_to": "external_design"}
```

---

## 6. teraflow group propose

AI分析により要件をグループに分割する提案を生成する。

```
ユーザー: teraflow group propose
    │
    ▼
[cmd/group/propose.go]
    │
    ▼
[core/group.go] GroupManager.Propose()
    │
    ├──(1) 要件ファイル収集
    │      [store/frontmatter.go] → docs/ 配下のMarkdownファイルを列挙
    │      frontmatterのnode_idがreq:で始まるファイルを抽出
    │
    ├──(2) AI分析
    │      [ai/prompt/builder.go] → group_propose.tmpl にデータ注入
    │      [ai/provider.go] → Provider.Complete()
    │      入力: 要件ファイルの内容 + 依存関係
    │      出力: グループ分割案（YAML形式）
    │
    ├──(3) 提案表示
    │      ユーザーにグループ案をテーブル表示
    │      各グループ: 名前、担当スコープ（ディレクトリ）、推奨人数、依存関係
    │
    └──(4) ユーザー確認後 → teraflow group define で正式定義
```

---

## 7. teraflow schedule predict

スケジュール完了予測を算出する。

```
ユーザー: teraflow schedule predict
    │
    ▼
[cmd/schedule/predict.go]
    │
    ▼
[core/schedule.go] ScheduleManager.Predict()
    │
    ├──(1) データ収集
    │      ├── [store] → project-state.yml (フェーズ履歴)
    │      ├── [store] → master-schedule.yml (計画値)
    │      ├── [store] → changelog/*.jsonl (変更ログ統計)
    │      └── [codd] → 影響分析結果（手戻り影響の考慮）
    │
    ├──(2) 予測計算
    │      ├── 方式1: 加重平均（過去フェーズの実績ベース）
    │      │   各フェーズの計画日数 vs 実績日数の比率を算出
    │      │   未完了フェーズの予測日数 = 計画日数 × 補正係数
    │      │
    │      └── 方式2: AI予測（--ai フラグ指定時）
    │          [ai/] → schedule_predict.tmpl
    │          入力: 全データ + 手戻り履歴
    │          出力: グループ別完了予測 + 信頼度
    │
    ├──(3) master-schedule.yml 更新
    │      predictions セクションを更新
    │
    └──(4) 結果表示
           グループ別の予測完了日 + 信頼度をテーブル表示
```

---

## 8. teraflow dashboard generate

プロジェクト状態のダッシュボードHTMLを生成する（10秒以内）。

```
ユーザー: teraflow dashboard generate [--format html|md]
    │
    ▼
[cmd/dashboard/generate.go]
    │
    ▼
[core/dashboard.go] DashboardGenerator.Generate()
    │
    ├──(1) データ収集（並列実行で高速化）
    │      ├── project-state.yml → ステージ・フェーズ状態
    │      ├── groups.yml → グループ一覧・メンバー
    │      ├── master-schedule.yml → スケジュール・予測
    │      ├── changelog/*.jsonl → 最近のアクティビティ
    │      └── codd/analyzer → 依存グラフ概要
    │
    ├──(2) サマリ生成（オプション: AI利用）
    │      [ai/] → dashboard_summary.tmpl
    │      ※ --no-ai で省略可（静的データのみで生成）
    │
    ├──(3) HTML/Markdown生成
    │      Go html/template でレンダリング
    │      ├── プロジェクト概要
    │      ├── ステージ進捗バー
    │      ├── グループ別フェーズ状態（マトリクス表示）
    │      ├── スケジュール予測グラフ（テキストベース）
    │      ├── 直近アクティビティ
    │      └── 手戻り・障害統計
    │
    └──(4) 出力
           .teraflow/dashboard/index.html に書き出し
           ※ Phase2で gh pages deploy による自動デプロイ
```

---

## 共通パターン

### エラーハンドリング

```
全コマンド共通:
1. .teraflow/ が存在しない → "Not a teraflow project. Run 'teraflow init' first."
2. YAML読み込みエラー → "Failed to load {file}: {error}"
3. gh コマンド失敗 → "GitHub operation failed: {error}. Is 'gh' installed and authenticated?"
4. AI API エラー → "AI provider error: {error}. Check your API key and network."
```

### 変更ログの一貫性

全てのデータ変更操作は変更ログ（JSONL）に記録する。
ログエントリの共通構造:

```json
{
  "type": "phase.advance",
  "timestamp": "2026-04-10T10:00:00Z",
  "actor": "user-a",
  "group": "group-auth",
  "details": { ... }
}
```

sec16で定義された30種のイベント型に準拠。

### SSoTフローの統一

```
[CLI操作] → [ローカルYAML更新] → [変更ログ記録] → [git commit] → [git push]
                                                                      ↓
                                                              [GitHub = SSoT]
```

git commit/push は自動実行しない（ユーザー操作に委ねる）。
`--auto-commit` フラグで自動コミットをオプション提供。
