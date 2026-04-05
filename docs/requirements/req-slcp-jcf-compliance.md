---
codd:
  node_id: "req:slcp-jcf-compliance"
  title: "要件定義: SLCP-JCF対応（共通フレーム準拠）"
  depends_on:
    - id: "req:overview"
      relation: extends
    - id: "req:non-functional"
      relation: extends
---

# 要件定義: SLCP-JCF対応（共通フレーム準拠）

## 1. 目的・スコープ

### 1.1 目的

teraflowを共通フレーム（SLCP-JCF2013, ISO/IEC 12207:2008準拠）に対応させ、以下を実現する:

1. 公共調達・企業導入における「共通フレーム準拠」の要件に対応可能とする
2. プロジェクトチーム全員がプロセスの現在状態を把握し、次に何をすべきかを即座に判断できるようにする
3. ロールベースのアクセス制御により、適切な権限を持つ者のみがプロセス遷移を実行できるようにする

### 1.2 スコープ（案B: 部分移行 + 可視化 + RBAC）

| 対象 | スコープ内 | スコープ外 |
|------|----------|----------|
| ドキュメント層 | 全docs/のTerasoluna参照を共通フレーム参照に更新 | -- |
| 概念対応表 | SLCP-JCFプロセス番号とteraflow概念の対応表を新規作成 | -- |
| CLIコマンド名 | 変更なし（stage/phase/gate等を維持） | stage→process等のリネーム |
| 新機能: プロセス可視化 | `teraflow process`コマンド新設 | -- |
| 新機能: ロール別表示 | `teraflow status --role=<role>`新設 | -- |
| 新機能: RBAC | teraflow.yml rbacセクション + 権限チェック | GitHub App認証（Phase3） |
| 新機能: ゲート承認 | `teraflow gate approve`コマンド新設 | -- |
| 新機能: 強制制約 | teraflow.yml constraintsセクション | -- |
| 新機能: 監査ログ | `.teraflow/audit-log.yml` | GitHub Actions連携（Phase2） |

### 1.3 適用方針

- ADR-010で決定した案B（部分移行）に基づく
- teraflow独自概念（rework, harness score, group, CoDD）は変更しない（差別化要素として維持）
- 既存CLIインタフェースに破壊的変更を加えない

## 2. teraflow概念とSLCP-JCFプロセスの対応表

### 2.1 ステージ対応

| teraflowステージ | SLCP-JCFプロセス | 番号 | 備考 |
|-----------------|-----------------|------|------|
| initial_development | システム開発 + ソフトウェア実装 | 2.3 + 2.4 | 複合プロセス |
| release | システム導入 + 受入れ支援 | 2.3.7 + 2.3.8 | -- |
| operation | 運用プロセス | 3.1 | 完全一致 |
| continuous_improvement | 保守プロセス（是正/適応/完全化保守） | 2.6 | 保守の一部 |
| maintenance | 保守プロセス（予防保守） | 2.6 | 保守の一部 |
| retirement | 廃棄プロセス | 3.2 | ほぼ一致 |

### 2.2 フェーズ対応

| teraflowフェーズ | SLCP-JCFアクティビティ | 番号 | 備考 |
|----------------|----------------------|------|------|
| requirements | 要件定義プロセス | 2.2 | 完全一致 |
| basic_design | ソフトウェア方式設計 | 2.4.3 | SI業界通称「基本設計」 |
| detailed_design | ソフトウェア詳細設計 | 2.4.4 | 完全一致 |
| implementation | ソフトウェア構築 | 2.4.5 | 構築=コーディング+単体テスト |
| testing | ソフトウェア結合テスト | 2.4.6 | -- |
| integration_test | ソフトウェア適格性確認テスト | 2.4.7 | -- |

### 2.3 機能概念対応

| teraflow概念 | SLCP-JCF対応 | 番号 | 備考 |
|-------------|-------------|------|------|
| gate | 共同レビュー / 妥当性確認 | 4.5 / 4.4 | teraflow拡張概念 |
| rework | 該当なし | -- | teraflow独自（差別化） |
| incident | 問題解決プロセス | 4.7 | 概念一致 |
| harness score | 測定プロセス | 5.8 | teraflow独自KPI |
| trace (CoDD) | 構成管理 + 検証 | 5.5 + 4.3 | teraflow独自実装 |
| group | 該当なし | -- | teraflow独自 |
| cycle | テーラリング | 第8区分 | 反復開発対応 |

## 3. プロセス状態追跡機能の要件

### 3.1 REQ-SLCP-001: project-state.yml拡張

project-state.ymlに`processes`セクションを追加し、各プロセスの状態を管理する。

**機能要件:**

- FR-001: 各プロセスは `completed`, `in_progress`, `not_started`, `skipped` の4状態を持つ
- FR-002: 各プロセスは `started_at`, `completed_at`, `gate_passed` フィールドを持つ
- FR-003: `teraflow stage advance` / `teraflow phase complete` 実行時に対応するプロセスの状態を自動更新する
- FR-004: 既存の `lifecycle.current_stage` / `phases.current` フィールドは変更しない（後方互換）
- FR-005: processesセクションが存在しないproject-state.ymlでも既存コマンドは正常動作する

### 3.2 REQ-SLCP-002: `teraflow process` コマンド

プロセス全体の進捗を可視化するコマンドを新設する。

**機能要件:**

- FR-010: `teraflow process` で全プロセスの状態一覧を表示する
- FR-011: プログレスバーで全体進捗率を表示する（完了プロセス数 / 全プロセス数）
- FR-012: 各プロセスにSLCP-JCFプロセス番号を併記する
- FR-013: 次のゲート条件と充足状況を表示する
- FR-014: `teraflow process show <name>` で特定プロセスの詳細を表示する
- FR-015: `teraflow process approve <name>` でプロセスのゲート承認を行う（RBAC連携）
- FR-016: `--output=json` フラグでJSON出力に対応する

### 3.3 REQ-SLCP-003: ゲート条件の自動評価

**機能要件:**

- FR-020: teraflow.ymlの`gate_rules`セクションでプロセスごとの完了条件を定義可能とする
- FR-021: 条件タイプ: `document_exists`（ファイル存在）, `test_pass_rate`（テスト通過率）, `coverage`（カバレッジ）, `manual_approval`（手動承認）
- FR-022: Phase2向け条件タイプ（予約）: `issue_label`, `pr_merged`, `issue_ratio`
- FR-023: 条件の自動評価結果を `teraflow process` および `teraflow process gate <name>` で表示する
- FR-024: 全条件充足時のみゲート通過可能とする（`manual_approval` は別途承認操作が必要）

## 4. ロール別「次にやるべきこと」表示の要件

### 4.1 REQ-SLCP-004: `teraflow status --role` コマンド

**機能要件:**

- FR-030: `teraflow status --role=<pm|dev|qa|stakeholder>` でロール別の次アクション表示
- FR-031: PM向け: ゲート承認待ち、リワーク未解決、次フェーズ準備事項を表示
- FR-032: 開発者向け: 現フェーズの未完了作業、リワーク指摘、次フェーズの事前確認を表示
- FR-033: QA向け: テスト計画状況、品質メトリクス（手戻り率、ハーネススコア）、未レビュー件数を表示
- FR-034: ステークホルダー向け: マイルストーン一覧、全体進捗率、次マイルストーン予定を表示
- FR-035: ロール未指定時は全ロールのサマリを表示する
- FR-036: `--output=json` フラグでJSON出力に対応する

### 4.2 データソース（Phase1）

Phase1ではローカルデータのみを使用する:
- project-state.yml（プロセス状態）
- rework-log.yml（リワーク状況）
- incident-log.yml（障害状況）
- `go test` 結果（テスト通過率）
- `coverage.out`（カバレッジ）

Phase2でGitHub API（Issues, PR, Actions）データを追加する。

## 5. RBAC権限管理の要件

### 5.1 REQ-SLCP-005: ロールベースアクセス制御

**機能要件:**

- FR-040: teraflow.ymlの`rbac`セクションでロール定義・メンバー・権限を設定可能とする
- FR-041: 定義可能ロール: `pm`, `architect`, `developer`, `qa`, `release_mgr`, `stakeholder`
- FR-042: 権限はリソース.アクション.ターゲットの3階層（例: `gate.approve.detailed_design`）
- FR-043: ワイルドカード対応: `gate.approve.*`（全ゲート承認可）、`*.read`（全閲覧可）
- FR-044: `rbac.enabled: false`（デフォルト）の場合は全操作を許可する（後方互換）
- FR-045: Phase1のユーザー識別は `git config user.name` または `--user` フラグで行う

### 5.2 REQ-SLCP-006: ゲート承認権限

**機能要件:**

- FR-050: フェーズ遷移にゲート承認を必要とするルールを定義可能とする
- FR-051: 承認に必要なロールをフェーズ遷移ごとに設定可能とする
- FR-052: 複数承認者対応: `quorum: all`（全員）、`majority`（過半数）、`any`（1名以上）
- FR-053: 権限不足時は明確なエラーメッセージを表示する（必要ロール、現在のロールを含む）

### 5.3 REQ-SLCP-007: 操作の強制制約

**機能要件:**

- FR-060: teraflow.ymlの`constraints`セクションで制約ルールを定義可能とする
- FR-061: 制約アクション: `block`（操作拒否）、`warn`（警告のみ）
- FR-062: 制約条件: テスト通過率、カバレッジ、リワーク未解決数、障害件数、ファイル存在
- FR-063: `--force` フラグで制約をオーバーライド可能とする（PMまたはrelease_mgrロール限定）
- FR-064: `--force` 使用時は `--reason` フラグによる理由記載を必須とする
- FR-065: `--force` 使用は監査ログに記録する

### 5.4 REQ-SLCP-008: 監査ログ

**機能要件:**

- FR-070: `.teraflow/audit-log.yml` にゲート承認・ステージ遷移・権限拒否・制約オーバーライドを記録する
- FR-071: 各エントリに timestamp, user, role, action, target, result を含める
- FR-072: `teraflow audit list` で監査ログを表示する
- FR-073: `--user`, `--action`, `--since` フィルタに対応する
- FR-074: `--output=json` フラグでJSON出力に対応する

## 6. 非機能要件

### 6.1 後方互換性（最重要）

- NFR-001: 既存CLIコマンド（stage, phase, rework, incident等）のインタフェースを変更しない
- NFR-002: processesセクションのないproject-state.ymlで既存コマンドが正常動作する
- NFR-003: rbac.enabled未設定時は全操作を許可する
- NFR-004: constraints未設定時は制約なしで動作する

### 6.2 パフォーマンス

- NFR-010: `teraflow process` の応答時間は500ms以内（ローカルデータのみの場合）
- NFR-011: `teraflow status --role=<role>` の応答時間は500ms以内

### 6.3 テスト

- NFR-020: 新規コマンドのユニットテストカバレッジ80%以上
- NFR-021: 既存テストが全件PASSし続けること（回帰テスト）

### 6.4 ドキュメント

- NFR-030: 全新規コマンドの`--help`テキストを日英併記する
- NFR-031: docs/guide/slcp-jcf-mapping.md で対応表をユーザーに提供する

## 7. 受入条件（Definition of Done）

### 7.1 ドキュメント層更新

- [ ] docs/guide/concepts.md のTerasoluna参照を共通フレーム参照に更新
- [ ] docs/guide/slcp-jcf-mapping.md（対応表）を新規作成
- [ ] 関連ADR・設計書のTerasoluna参照を更新（11ファイル）
- [ ] codd scan が正常完了する

### 7.2 プロセス可視化

- [ ] `teraflow process` コマンドが動作する
- [ ] `teraflow process show <name>` が動作する
- [ ] `teraflow process approve <name>` が動作する（RBAC連携）
- [ ] プログレスバー + SLCP-JCFプロセス番号が表示される
- [ ] `--output=json` が動作する

### 7.3 ロール別表示

- [ ] `teraflow status --role=pm` が動作する
- [ ] `teraflow status --role=dev` が動作する
- [ ] `teraflow status --role=qa` が動作する
- [ ] `teraflow status --role=stakeholder` が動作する

### 7.4 RBAC・権限管理

- [ ] teraflow.yml rbacセクションが機能する
- [ ] 権限不足時にエラーメッセージが表示される
- [ ] `teraflow gate approve` が権限チェック付きで動作する
- [ ] constraints制約がblock/warnで機能する
- [ ] `--force` + `--reason` でオーバーライドが動作する
- [ ] 監査ログが`.teraflow/audit-log.yml`に記録される
- [ ] `teraflow audit list` が動作する

### 7.5 品質

- [ ] 既存テスト全件PASS（回帰テスト）
- [ ] 新規コマンドのカバレッジ80%以上
- [ ] CI全緑
