---
codd:
  node_id: "design:phase2-impl-plan"
  title: "Phase2 実装計画 — Wave分割とタスク分解"
  depends_on:
    - id: "design:phase2-system"
      relation: derives_from
    - id: "adr:005-github-actions-design"
      relation: implements
    - id: "adr:006-event-driven"
      relation: implements
    - id: "adr:007-non-engineer-ui"
      relation: implements
    - id: "adr:008-phase1-coexistence"
      relation: implements
    - id: "design:cicd-workflows"
      relation: extends
---

# Phase2 実装計画 — Wave分割とタスク分解

## 1. 目的・スコープ

### 1.1 Phase2の目的

Phase1（ローカルCLI、エンジニア向け）に **GitHub.com連携層** を追加し、以下を実現する:

1. **非エンジニア（PM/PMO）がブラウザからプロジェクト管理を行える** — Issueテンプレート、ラベル、Discussion、GitHub Projects活用（ADR-007）
2. **GitHub Actionsによるイベント駆動自動化** — Issue/PR/Discussionイベントをトリガーにteraflow CLIを自動実行（ADR-005, ADR-006）
3. **CLI ↔ Actions間のデータ整合性確保** — 楽観的並行制御 + concurrencyグループ（ADR-008, W-005）
4. **AI Agent Pipeline** — 10種のAI Agentによる実装/レビュー/CI修正/要件整理の自動化

### 1.2 スコープ

| 含む | 含まない |
|------|---------|
| 16 Actionsワークフロー（TF-001〜016） | GitHub App開発（Phase3） |
| 6 Issueテンプレート | SaaS Web UI（Phase4） |
| 3 Discussionカテゴリテンプレート | マルチリポ対応（Phase3） |
| GitHub Pages ダッシュボードデプロイ | DB移行（Phase4） |
| internal/actions/ パッケージ | 独自認証基盤 |
| internal/agent/ パッケージ | Marketplace有料プラン掲載 |
| `teraflow setup actions/templates` コマンド拡張 | |

### 1.3 前提条件

- Phase1 CLI が安定版（v0.4.0+、テストカバレッジ85%、E2E全PASS）
- GoReleaser によるリリースバイナリが利用可能（CD-001）
- GitHub Actions CI/CD（CI-001〜006, QA-001〜003）が稼働中
- codd依存グラフ: 47 nodes, 90 edges（Phase2設計完了時点）

---

## 2. 16ワークフロー一覧と役割

### 2.1 カテゴリ分類

| カテゴリ | ワークフロー | 特徴 |
|---------|------------|------|
| A: ゲート・遷移系（コア） | TF-001, TF-002, TF-003, TF-005 | state更新を伴う。Phase2の根幹 |
| B: AI Agent系 | TF-004, TF-007, TF-008, TF-009, TF-010, TF-014 | AI API呼び出し。internal/agent/ 依存 |
| C: 定期実行・自動記録系 | TF-006, TF-011, TF-012, TF-013, TF-015, TF-016 | cron/イベント駆動の自動処理 |

### 2.2 依存関係グラフ

```
Wave 1 (基盤):
  internal/actions/ (共通構造)
  Issueテンプレート (6種)
  Discussionカテゴリ (3種)
  teraflow setup actions/templates 拡張
      │
      ▼
Wave 2 (コア操作):
  TF-003 (phase-transition) ← Issueテンプレート依存
  TF-001 (phase-gate) ← TF-003のstate更新に依存
  TF-002 (permission-guard)
  TF-005 (artifact-finalize)
  TF-006 (changelog-update)
  TF-012 (rework-impact)
      │
      ▼
Wave 3 (AI Agent):
  internal/agent/ (AgentManager)
  TF-004 (req-agent)
  TF-009 (review-agent)
  TF-008 (ci-fix-agent)
  TF-007 (implement-agent)
  TF-010 (conflict-agent)
  TF-014 (incident-agent)
      │
      ▼
Wave 4 (定期実行・ダッシュボード):
  TF-013 (dashboard-deploy) ← GitHub Pages設定依存
  TF-011 (stuck-monitor)
  TF-015 (maintenance-agent)
  TF-016 (schedule-predict)
```

---

## 3. Wave分割

### Wave選定基準

1. **依存関係**: 下流Waveが上流の成果物を前提とする場合は先行
2. **価値提供**: 非エンジニアが最も早く恩恵を受けるものを優先
3. **リスク**: CLIバイナリ呼び出し・state更新の基盤が安定してからAI Agent系を構築
4. **テスト容易性**: 独立してE2Eテスト可能な粒度で分割

---

### Wave 1: 基盤構築（Issueテンプレート + Actions共通基盤）

**目標**: Phase2の土台を作る。非エンジニアが操作できるIssueテンプレートと、全ワークフロー共通のActions基盤を整備。

**完了条件**:
- `teraflow setup actions` で16ワークフローYAMLがリポジトリに生成される
- `teraflow setup templates` で6 Issueテンプレート + 3 Discussionカテゴリが生成される
- Actions内でteraflow CLIバイナリのインストール・実行が成功する（POC実証）
- concurrency グループが正常に動作する（2ワークフロー同時起動で逐次実行を確認）

| タスクID | 内容 | 見積 | 依存 | 担当候補 |
|---------|------|------|------|---------|
| W1-01 | `internal/actions/` パッケージ作成 — embed.FSでワークフローテンプレート管理、GenerateWorkflows/ValidateWorkflow実装 | L | なし | ashigaru |
| W1-02 | 16ワークフローYAMLテンプレート作成 — 共通構造（checkout, install teraflow, pull, execute, commit+push+retry）をベースに16ファイル | L | W1-01 | ashigaru |
| W1-03 | `cmd/setup_actions.go` 拡張 — `teraflow setup actions` で `.github/workflows/teraflow-*.yml` を生成 | M | W1-01 | ashigaru |
| W1-04 | 6 Issueテンプレート作成（YAML form形式） — フェーズ開始/完了/スキップ/手戻り/インシデント/Agent実装依頼 | M | なし | ashigaru |
| W1-05 | 3 Discussionカテゴリテンプレート作成 — 要件議論/設計議論/振り返り | S | なし | ashigaru |
| W1-06 | `cmd/setup_templates.go` 拡張 — `teraflow setup templates` でテンプレート生成 | M | W1-04, W1-05 | ashigaru |
| W1-07 | Actions基盤POC — teraflowバイナリのインストール・実行・concurrency動作確認（手動テスト） | M | W1-02 | ashigaru |
| W1-08 | ユニットテスト（internal/actions/）+ E2Eテスト（setup actions/templates） | M | W1-01〜06 | ashigaru |

**Wave 1 合計**: S×1, M×5, L×2（推定8タスク）

---

### Wave 2: コア操作ワークフロー（ゲート・遷移・自動記録）

**目標**: PM/PMOがブラウザからフェーズ管理・手戻り・changelog記録を行えるようにする。Phase2の最大価値提供。

**完了条件**:
- Issue作成（テンプレート）→ フェーズ遷移が自動実行される
- PRマージ → changelog自動追記される
- ラベル付与 → ステージゲートチェックが実行される
- 手戻りIssue → 影響分析が自動実行される
- W-005: concurrencyグループによるActions間排他制御が機能する

| タスクID | 内容 | 見積 | 依存 | 担当候補 | ワークフロー |
|---------|------|------|------|---------|------------|
| W2-01 | TF-003 phase-transition.yml 実装 — Issueテンプレートbody解析、phase start/complete/skip自動実行、Issueコメント結果通知 | L | W1-04 | ashigaru | TF-003 |
| W2-02 | TF-001 phase-gate.yml 実装 — PR時のフェーズ/ステージ整合性チェック、NG時PRコメント | M | W2-01 | ashigaru | TF-001 |
| W2-03 | TF-002 permission-guard.yml 実装 — CODEOWNERS+ロール権限チェック、権限不足時レビュー要求 | M | なし | ashigaru | TF-002 |
| W2-04 | TF-005 artifact-finalize.yml 実装 — 確定ラベル検知→成果物ファイルリンク生成+カウント更新 | M | W2-01 | ashigaru | TF-005 |
| W2-05 | TF-006 changelog-update.yml 実装 — PRマージ時タイトル解析→changelog JSONL追記 | M | なし | ashigaru | TF-006 |
| W2-06 | TF-012 rework-impact.yml 実装 — rework-*ラベル検知→teraflow rework create→影響分析結果をIssueコメント | M | W2-01 | ashigaru | TF-012 |
| W2-07 | W-005 リトライ機構のshared script化 — `scripts/push-with-retry.sh` として共通化、全ワークフローから呼び出し | S | W2-01 | ashigaru | 共通 |
| W2-08 | E2Eテスト（Wave 2） — テストリポジトリでIssue作成→Actions起動→state更新の一連フロー確認 | L | W2-01〜06 | ashigaru |

**Wave 2 合計**: S×1, M×4, L×2（推定8タスク）

---

### Wave 3: AI Agent Pipeline

**目標**: 10種のAI Agentを統合管理するinternal/agent/を構築し、6つのAgent系ワークフローを実装する。

**完了条件**:
- `teraflow agent assign --type <type>` で各AgentTypeが実行可能
- Discussion投稿 → AI要件整理Agentが応答
- PR作成 → AIレビューAgentが自動レビュー
- CI失敗 → 修正提案Agentが起動
- Agent実行結果がIssueコメント/PRコメントとして出力される

| タスクID | 内容 | 見積 | 依存 | 担当候補 | ワークフロー |
|---------|------|------|------|---------|------------|
| W3-01 | `internal/agent/` パッケージ作成 — AgentManager、AgentType定義、AgentContext/AgentResult型、Provider Interface連携 | L | なし | ashigaru |  |
| W3-02 | Agent共通プロンプトテンプレート — embed.FSで管理。各AgentType別のシステムプロンプト+コンテキスト注入テンプレート | M | W3-01 | ashigaru |  |
| W3-03 | `cmd/agent.go` 拡張 — `teraflow agent assign/status` コマンド実装 | M | W3-01 | ashigaru |  |
| W3-04 | TF-009 review-agent.yml 実装 — PR opened/synchronize → AIレビュー実行 → PRコメント投稿 | M | W3-01, W3-02 | ashigaru | TF-009 |
| W3-05 | TF-004 req-agent.yml 実装 — Discussion要件議論カテゴリ → AI要件整理 → Discussionコメント | M | W3-01, W3-02 | ashigaru | TF-004 |
| W3-06 | TF-008 ci-fix-agent.yml 実装 — check_run failure → ログ解析 → 修正PR自動作成 or Issue提案 | L | W3-01, W3-02 | ashigaru | TF-008 |
| W3-07 | TF-007 implement-agent.yml 実装 — agent-implementラベル → 実装Agent起動 → PR自動作成 | L | W3-01, W3-02 | ashigaru | TF-007 |
| W3-08 | TF-010 conflict-agent.yml 実装 — conflictラベル → コンフリクト解消 → PR更新 | M | W3-01 | ashigaru | TF-010 |
| W3-09 | TF-014 incident-agent.yml 実装 — incident-*ラベル → 障害調査Agent → 調査結果Issueコメント | M | W3-01, W3-02 | ashigaru | TF-014 |
| W3-10 | ユニットテスト（internal/agent/）+ AgentManager統合テスト | M | W3-01〜03 | ashigaru |  |

**Wave 3 合計**: M×7, L×3（推定10タスク）

---

### Wave 4: 定期実行・ダッシュボード・仕上げ

**目標**: 定期実行ワークフローとGitHub Pagesダッシュボードを稼働させ、Phase2を完成する。

**完了条件**:
- GitHub Pagesでダッシュボードが自動デプロイ・更新される
- 平日9時に停滞Issue監視が実行される
- 週次で保守スコアリング・スケジュール予測が実行される
- Phase2全ワークフロー16本がテストリポジトリで稼働確認される

| タスクID | 内容 | 見積 | 依存 | 担当候補 | ワークフロー |
|---------|------|------|------|---------|------------|
| W4-01 | TF-013 dashboard-deploy.yml 実装 — push/daily → `teraflow dashboard generate` → GitHub Pages デプロイ設定 | L | なし | ashigaru | TF-013 |
| W4-02 | TF-011 stuck-monitor.yml 実装 — 平日9時cron → 停滞Issue検出 → 通知Issue作成 | M | なし | ashigaru | TF-011 |
| W4-03 | TF-015 maintenance-agent.yml 実装 — 週次日曜cron → harness score → 閾値超過でIssue起票 | M | W3-01 | ashigaru | TF-015 |
| W4-04 | TF-016 schedule-predict.yml 実装 — 週次月曜cron → schedule predict → schedule.yml更新 | M | なし | ashigaru | TF-016 |
| W4-05 | GitHub Projects連携設定ドキュメント — カンバンボード設定手順、自動移動ルール | S | W2-01 | gunshi |  |
| W4-06 | Phase2 全体E2Eテスト — テストリポジトリで16ワークフロー統合テスト | L | W4-01〜04 | ashigaru |  |
| W4-07 | Phase2 ユーザーガイド — 非エンジニア向け操作マニュアル（Issueテンプレートの使い方、ラベル操作等） | M | W4-06 | ashigaru |  |
| W4-08 | Phase2 リリースノート + バージョンアップ（v1.0.0） | S | W4-06 | ashigaru |  |

**Wave 4 合計**: S×2, M×4, L×2（推定8タスク）

---

## 4. タスクサマリ

### 4.1 全体規模

| Wave | タスク数 | S | M | L | 主な成果物 |
|------|---------|---|---|---|----------|
| Wave 1 | 8 | 1 | 5 | 2 | internal/actions/, テンプレート, setup拡張 |
| Wave 2 | 8 | 1 | 4 | 2 | TF-001〜006, TF-012, リトライ共通化 |
| Wave 3 | 10 | 0 | 7 | 3 | internal/agent/, TF-004〜010, TF-014 |
| Wave 4 | 8 | 2 | 4 | 2 | TF-011〜016, E2E, ドキュメント |
| **合計** | **34** | **4** | **20** | **9** | |

### 4.2 見積目安

| サイズ | 足軽1名の作業量 | 目安時間 |
|-------|--------------|---------|
| S | 1セッションで完了 | コンテキスト1回分 |
| M | 1〜2セッション | コンテキスト1〜2回分 |
| L | 2〜3セッション | コンテキスト2〜3回分。分割推奨 |

### 4.3 並列実行可能性

| Wave | 並列度 | 備考 |
|------|-------|------|
| Wave 1 | W1-01〜03 と W1-04〜06 は並列可 | Actions基盤とテンプレートは独立 |
| Wave 2 | W2-02,03,05 は W2-01完了後に並列可 | TF-003が多くの下流の前提 |
| Wave 3 | W3-04〜09 は W3-01〜03完了後に並列可 | Agent基盤完成後、各Agent並列実装 |
| Wave 4 | W4-01〜04 は並列可 | 各ワークフローは独立 |

---

## 5. codd-devフロー適用方針

### 5.1 Phase2でのcodd-dev利用

Phase2の設計書・実装計画にもcodd frontmatterを付与し、依存グラフを維持する。

| フェーズ | codd操作 | 期待ノード数 |
|---------|---------|------------|
| Phase2設計完了（現在） | scan済み | 47 nodes, 90 edges |
| Wave 1完了 | scan（新規: internal/actions/ 設計メモ） | ~49 nodes |
| Wave 2完了 | scan（ワークフロー設計メモ追加） | ~52 nodes |
| Wave 3完了 | scan（internal/agent/ 設計メモ） | ~55 nodes |
| Wave 4完了 | scan + impact確認 | ~58 nodes, ~120 edges |

### 5.2 Wave間のcodd scan実施タイミング

各Wave完了時に `codd scan` を実施し、依存グラフの整合性を検証する。

```
Wave完了 → codd scan → NG(orphan/broken link) → 修正 → 再scan → OK → 次Wave開始
```

### 5.3 新規ファイルのcodd frontmatter方針

- **設計ドキュメント**: node_id + depends_on 必須
- **ワークフローYAML**: frontmatter不要（GitHub Actions固有形式）
- **Issueテンプレート**: frontmatter不要（GitHub固有形式）
- **Go ソースコード**: frontmatter不要（コードは依存グラフの対象外）

---

## 6. リスクと対策

### 6.1 Phase1とPhase2の共存（ADR-008）

| リスク | 影響 | 対策 |
|-------|------|------|
| CLI操作中にActionsが同時にstate更新 | project-state.yml競合 | concurrencyグループ + 3回retry + Issueコメント通知 |
| Phase1 CLI v0.4.x とPhase2 Actions内バイナリのバージョン不一致 | コマンド挙動の不整合 | Actions内は常にlatest releaseをダウンロード。バイナリバージョンをActions logに出力 |
| Phase1テストがPhase2コード追加で壊れる | リグレッション | 既存CI（CI-001〜006）が全Phaseで常時稼働。Phase2コードは新パッケージに隔離 |

### 6.2 非エンジニア向けUI導入（ADR-007）

| リスク | 影響 | 対策 |
|-------|------|------|
| Issueテンプレートのドロップダウンがgroups.ymlと同期しない | 存在しないグループを選択 | TF-003内でグループ存在チェック。不一致時はIssueコメントでエラー通知 |
| ラベル名の変更がActionsトリガーを壊す | ワークフロー不発 | ラベル名はteraflow.ymlで定義。`teraflow doctor` でラベル不整合チェックを追加 |
| 非エンジニアがActions失敗ログを読めない | サポート問い合わせ増加 | 全ワークフローでIssueコメントに分かりやすい成功/失敗メッセージを出力 |

### 6.3 AI Agent系のリスク

| リスク | 影響 | 対策 |
|-------|------|------|
| AI API費用の予測困難 | コスト超過 | Agent実行回数をteraflow.ymlで制限（月次上限設定）。ログで使用量追跡 |
| AI生成コードの品質が不安定 | 低品質PR | Agent信頼度（supervised）: 人間レビュー必須をデフォルトに。自動マージ禁止 |
| AI APIレートリミット | Agent実行失敗 | リトライ + exponential backoff。キュー管理はconcurrencyグループで代替 |

### 6.4 全体リスク

| リスク | 影響 | 対策 |
|-------|------|------|
| 16ワークフローの同時開発による品質低下 | バグ/不整合 | Wave分割による段階的リリース。各Wave完了時にE2Eテスト |
| GitHub Actions実行時間制限（6時間/job） | AI Agent処理がタイムアウト | Agent処理はworker_timeout設定で制御。長時間タスクは分割実行 |
| テストリポジトリの維持コスト | テスト環境の陳腐化 | テストリポジトリの自動セットアップスクリプトを用意 |

---

## 7. 成功指標（KPI）

### 7.1 Wave別KPI

| Wave | KPI | 検証方法 |
|------|-----|---------|
| Wave 1 | `teraflow setup actions` で16 YAMLが生成される | E2Eテスト: ファイル存在+YAML構文チェック |
| Wave 1 | `teraflow setup templates` で6テンプレート+3カテゴリが生成される | E2Eテスト: ファイル存在+YAML構文チェック |
| Wave 1 | Actions内でCLIバイナリが正常実行される | テストリポジトリでPOC実行 |
| Wave 2 | Issue作成→フェーズ遷移が60秒以内に完了 | テストリポジトリでタイマー計測 |
| Wave 2 | PRマージ→changelog追記が自動実行される | テストリポジトリでPRマージ後にJSONL確認 |
| Wave 2 | concurrencyグループで同時実行が防止される | 2つのIssueを同時クローズしてActions実行ログ確認 |
| Wave 3 | PR作成→AIレビューコメントが投稿される | テストリポジトリでPR作成後にコメント確認 |
| Wave 3 | Discussion投稿→AI応答が投稿される | テストリポジトリでDiscussion作成後にコメント確認 |
| Wave 4 | GitHub Pagesにダッシュボードが自動デプロイされる | テストリポジトリでpush後にPages確認 |
| Wave 4 | 16ワークフロー全てがテストリポジトリで稼働 | 統合E2Eテスト全PASS |

### 7.2 Phase2全体KPI

| 指標 | 目標値 |
|------|-------|
| テストカバレッジ（Phase2新規コード） | 80%以上 |
| E2Eテストシナリオ数 | 20以上（Phase1の31に加えてPhase2固有20） |
| ワークフローYAML lint PASS率 | 100% |
| 非エンジニアの操作成功率（手動テスト） | 90%以上（テンプレート入力→正常完了） |
| codd依存グラフ ノード数 | 55以上 |
| Phase1既存テスト維持 | 全PASS（リグレッションなし） |

---

## 8. 設計レビュー（自己確認結果）

### 8.1 ADR-005〜008との整合性 ✅

| ADR | 整合性 | 確認内容 |
|-----|--------|---------|
| ADR-005 (Actions設計) | ✅ | CLIバイナリ再利用パターンを全ワークフローで採用。リリースバイナリをダウンロード |
| ADR-006 (イベント駆動) | ✅ | on: issues/pull_request/discussion/check_run/scheduleを使用。concurrency group統一 |
| ADR-007 (非エンジニアUI) | ✅ | Wave 1でIssueテンプレート6種+Discussionカテゴリ3種を先行整備 |
| ADR-008 (Phase1共存) | ✅ | W-005対応: リトライ共通化(W2-07)、concurrency設計、CLIとの競合対応文書化 |

### 8.2 Wave分割の依存関係矛盾チェック ✅

- Wave 1 → Wave 2: テンプレートとActions基盤が先行（問題なし）
- Wave 2 → Wave 3: コア操作が安定してからAI Agent構築（問題なし）
- Wave 3 → Wave 4: Agent基盤がTF-015(maintenance-agent)で利用される（依存正しい）
- Wave 4内: W4-01〜04は並列可、W4-06(E2E)は全完了後（問題なし）

### 8.3 タスク粒度の適正性 ✅

- S: 1セッション完結。テンプレート作成、ドキュメント、リリースノート
- M: 1〜2セッション。個別ワークフロー実装（YAML+テスト）
- L: 2〜3セッション。パッケージ新規作成、E2Eテスト。必要に応じてさらに分割可能

### 8.4 Phase1との後方互換性 ✅

- Phase2コードは `internal/actions/` と `internal/agent/` に隔離。既存パッケージを変更しない
- `cmd/setup_actions.go` と `cmd/setup_templates.go` は新規追加。既存コマンドに影響なし
- Phase1テスト（85%カバレッジ、E2E 31チェック）は全て維持される
- teraflow.yml の設定互換: Phase2設定はオプショナル項目として追加（Phase1設定は変更なし）
