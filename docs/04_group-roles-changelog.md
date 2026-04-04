---
codd:
  node_id: "req:group-roles-changelog"
  title: "グループ管理・ロール権限・変更ログ運用"
  depends_on:
    - id: "req:teraflow-overview"
      relation: derives_from
    - id: "req:cli-project-mgmt"
      relation: derives_from
    - id: "req:github-labels-issues"
      relation: derives_from
---

## 15. グループ管理

### 15.1 グループの概念

プロジェクトを責務・技術領域・チームで分割した単位。各グループは独立してフェーズを進行でき、同期ポイントで必要に応じて合わせる。

### 15.2 グループのライフサイクル

```
未定義     要求整理〜要件定義前半はグループなし（shared のみ）
  │
  ▼  teraflow group propose（要件定義中盤）
仮定義     要件にグループタグを付けながら定義を継続
  │
  ▼  teraflow group finalize（要件定義→基本設計ゲート）
確定       基本設計以降はグループごとに独立進行
  │
  ▼  teraflow group rebalance（必要に応じて随時）
再編成     人員変動時にグループ構成を見直し
```

### 15.3 グループ提案 (`teraflow group propose`)

確定済みの要件から技術領域分布を分析し、人員配置案と組み合わせてグループ分割案を自動提示する。

```
$ teraflow group propose --team-size 12

確定済み要件: 32件

技術領域分析:
  画面系:      12件 (38%)
  API/ロジック: 10件 (31%)
  データ:       5件 (16%)
  インフラ:     3件 (9%)
  横断:         2件 (6%)

─── 提案A: 4グループ ───
  frontend (4名), backend (4名), infra (2名), shared (2名)

─── 提案B: 3グループ ───
  frontend (4名), backend (5名), shared (3名)

どの案を採用しますか？
```

### 15.4 グループ確定のゲート条件

要件定義→基本設計のフェーズ遷移ゲートに、グループ確定を条件として含める。

```yaml
# .github/teraflow.yml（phases セクション）

phases:
  要件定義:
    gate_conditions:
      - type: files_exist
        path: "docs/shared/02_requirement-definition/REQDEF-*.md"
      - type: no_open_issues
        label: "phase:要件定義"
      - type: gate_issue_closed
        issue_label: "gate:要件定義"
      - type: groups_finalized           # グループ確定済みであること
      - type: all_artifacts_assigned      # 全要件にグループが割り当て済み
```

### 15.5 同期ポイント

グループ間の依存関係を管理する仕組み。特定のフェーズに進むために、他グループが特定の状態にあることを要求する。

```yaml
# .github/groups.yml（sync_points セクション）

sync_points:
  - id: "SP-001"
    name: "API IF 合意"
    required_groups: ["backend", "frontend"]
    required_phase: "要件定義"
    blocks:
      - group: "frontend"
        phase: "基本設計"

  - id: "SP-002"
    name: "インフラ環境準備完了"
    required_groups: ["infra"]
    required_phase: "実装"
    blocks:
      - group: "backend"
        phase: "テスト"
      - group: "frontend"
        phase: "テスト"
```

### 15.6 グループ別のフェーズ独立性

```
基本は独立、ただし同期ポイントで制約:

  各グループが自分のゲート条件を満たせば advance できる。
  他グループの状態は関係ない。

  ただし同期ポイントで定義された制約がある場合、
  指定されたグループが指定フェーズを完了するまでブロックされる。
```

### 15.7 手戻りのグループ別影響制御

手戻り発生時、影響分析をグループ別に分類する。影響のないグループは巻き込まない。

```
手戻り影響分析（グループ別）:

  ■ backend（発生元）
    現フェーズ: 詳細設計 → 要件定義に戻す必要あり
    影響成果物: 4件

  ■ frontend
    グループ横断影響: あり（API IF に依存）
    影響成果物: 2件
    → 基本設計の一部を再レビュー必要

  ■ infra
    グループ横断影響: なし
    → 影響なし、現フェーズを継続
```

### 15.8 グループ定義ファイル (`groups.yml`)

```yaml
# .github/groups.yml

status: "finalized"        # draft / finalized
finalized_at: "2026-05-09"
finalized_by: "@pm-tanaka"

groups:
  - id: frontend
    display_name: "フロントエンド"
    docs_path: "docs/frontend"
    phases: [要求整理, 要件定義, 基本設計, 詳細設計, 実装, テスト]

  - id: backend
    display_name: "バックエンド"
    docs_path: "docs/backend"
    phases: [要求整理, 要件定義, 基本設計, 詳細設計, 実装, テスト]

  - id: infra
    display_name: "インフラ"
    docs_path: "docs/infra"
    phases: [要求整理, 要件定義, 基本設計, 実装, テスト]

  - id: shared
    display_name: "横断・PMO"
    docs_path: "docs/shared"

members:
  # ロール定義・メンバー割り当てはセクション17で定義

sync_points:
  # 上記 15.5 参照
```

### 15.9 project-state.yml のグループ対応

```yaml
# .github/project-state.yml（groups セクション）

groups:
  backend:
    current_phase: "詳細設計"
    phase_history:
      - phase: "要求整理"
        started_at: "2026-04-01"
        ended_at: "2026-04-15"
        result: "completed"
      # ...

  frontend:
    current_phase: "基本設計"
    phase_history: [...]

  infra:
    current_phase: "実装"
    phase_history: [...]

sync_points_status:
  SP-001:
    status: "completed"
    completed_at: "2026-05-05"
  SP-002:
    status: "pending"
```

### 15.10 人員変動への対応 (`teraflow group rebalance`)

```
$ teraflow group rebalance --remove-member 2 --from backend

影響分析:
  backend 現在: 4名 → 2名
  残タスク: 8件（詳細設計3件 + 実装5件）
  予定遅延: +12日

  提案:
    A) backend の一部タスクを frontend に移管
    B) グループ統合（backend + frontend → app）
    C) スケジュール延伸を受容
```

---

## 16. 変更ログ（Changelog）

### 16.1 設計方針

全ファイルの変更履歴を集中管理する。各ファイルは「現在の状態」のみを保持し、「何がいつ変わったか」は changelog に記録する。

```
各ファイル（docs/, groups.yml 等）= スナップショット（現在の状態のみ）
.teraflow/changelog/YYYY-MM.jsonl  = イベントログ（追記のみ）
ダッシュボード                      = changelog を読みやすく表示
```

### 16.2 ファイル構成

```
.teraflow/
└── changelog/
    ├── 2026-04.jsonl       # 4月の変更ログ
    ├── 2026-05.jsonl       # 5月の変更ログ
    └── ...                 # 月ごとに自動分割
```

JSONL（1行1イベント）を採用する。追記のみでコンフリクトしにくく、構造化されているので検索・フィルタが容易。

### 16.3 イベントタイプ

| カテゴリ | type | 記録内容 |
|---|---|---|
| **フェーズ** | `phase.start` | フェーズ開始 |
| | `phase.advance` | フェーズ前進 |
| | `phase.revert` | フェーズ手戻り |
| | `phase.skip` | フェーズスキップ |
| **ステージ** | `stage.advance` | ステージ遷移 |
| | `stage.activate` | 並行ステージ有効化 |
| | `stage.freeze` | ステージ凍結 |
| **グループ** | `group.propose` | グループ仮定義 |
| | `group.finalize` | グループ確定 |
| | `group.rebalance` | グループ再編成 |
| **メンバー** | `member.join` | メンバー着任 |
| | `member.leave` | メンバー離任 |
| | `member.transfer` | グループ間移動 |
| | `member.role_change` | ロール変更 |
| **成果物** | `artifact.confirm` | 成果物ファイル確定 |
| | `artifact.update` | 成果物ファイル更新 |
| | `artifact.deprecate` | 成果物の廃止 |
| **手戻り** | `rework.create` | 手戻り起票 |
| | `rework.approve` | 手戻り承認 |
| | `rework.resolve` | 手戻り解決 |
| **障害** | `incident.create` | 障害報告 |
| | `incident.resolve` | 障害解決 |
| **スケジュール** | `schedule.predict` | 完了予測の更新 |
| | `schedule.update` | スケジュール実績記録 |
| **同期** | `sync.complete` | 同期ポイント完了 |
| **Agent** | `agent.assign` | Agent割り当て |
| | `agent.escalate` | 人間へエスカレーション |
| | `agent.complete` | Agent作業完了 |
| **権限** | `permission.denied` | 権限違反の検知 |

### 16.4 イベントレコードの例

```jsonl
{"ts":"2026-05-01T10:00:00+09:00","type":"member.assign","member":"@dev-a","group":"frontend","role":"developer","by":"@pm-tanaka"}
{"ts":"2026-06-01T09:00:00+09:00","type":"member.transfer","member":"@dev-c","from_group":"frontend","to_group":"backend","reason":"バックエンド人員不足","by":"@pm-tanaka"}
{"ts":"2026-06-15T15:30:00+09:00","type":"artifact.confirm","file":"docs/backend/04_detailed-design/DD-0040.md","phase":"詳細設計","group":"backend","by":"@dev-e","issue":40}
{"ts":"2026-05-20T14:00:00+09:00","type":"rework.create","issue":42,"detected_in":"基本設計","root_cause":"要件定義","group":"backend","by":"@dev-e"}
```

### 16.5 書き込みタイミング

自動書き込み（Actions）: 成果物確定、フェーズ遷移、手戻り承認、障害報告、スケジュール予測更新、権限違反検知時。

手動書き込み（CLI）: メンバー変更、グループ再編成、ステージ遷移時。

### 16.6 CLI での履歴検索 (`teraflow log show`)

```
$ teraflow log show --group backend --type member

  2026-05-01 10:00  member.assign    @dev-e → backend
  2026-05-01 10:00  member.assign    @dev-f → backend
  2026-06-01 09:00  member.transfer  @dev-c frontend→backend
  2026-06-15 17:00  member.leave     @dev-h ← backend
  2026-06-16 09:00  member.join      @dev-k → backend

$ teraflow log show --file "docs/backend/03_basic-design/BD-0030.md"

  2026-05-15 09:00  artifact.confirm  BD-0030.md 作成       by @dev-e
  2026-07-01 10:00  artifact.update   BD-0030.md 手戻り修正 by @dev-e
```

---

## 17. ロール・権限管理

### 17.1 ロール定義

| ロール | 名称 | 対象者 | 概要 |
|---|---|---|---|
| `requester` | 起票者 | 非エンジニア | 要求・要望を投稿する |
| `developer` | 開発者 | エンジニア | 設計・実装・テストを担当 |
| `reviewer` | レビュワー | シニアエンジニア | 成果物・コードの品質を承認 |
| `group_lead` | グループリード | リーダー | グループ内のフェーズ進行を管理 |
| `pm` | プロジェクトマネージャー | PM | プロジェクト全体の統制 |
| `admin` | 管理者 | システム管理者 | システム設定・権限管理 |

**包含関係**: `admin ⊃ pm ⊃ group_lead ⊃ reviewer ⊃ developer`。`requester` は独立。一人が複数ロールを持てる（例: backend の developer + frontend の reviewer）。

### 17.2 ロール × アクション マトリクス

凡例: ◎主担当 ○実行可 △自グループのみ ×不可

**要求・要件・設計フェーズ**:

| アクション | 起票者 | 開発者 | ﾚﾋﾞｭﾜー | Gﾘｰﾄﾞ | PM | Admin |
|---|---|---|---|---|---|---|
| Discussion投稿 | ◎ | ○ | ○ | ○ | ○ | ○ |
| AI対話（Discussion） | ◎ | ○ | ○ | ○ | ○ | ○ |
| 「要求確定」 | ◎ | × | × | ○ | ○ | ○ |
| Issue起票 | × | △ | △ | △ | ○ | ○ |
| AI対話（Issue） | × | △ | △ | △ | ○ | ○ |
| 「確定」 | × | × | △ | △ | ○ | ○ |
| 確定PR レビュー | × | × | ◎ | ○ | ○ | ○ |
| 確定PR マージ | × | × | ◎ | ○ | ○ | ○ |

**実装フェーズ**:

| アクション | 起票者 | 開発者 | ﾚﾋﾞｭﾜー | Gﾘｰﾄﾞ | PM | Admin |
|---|---|---|---|---|---|---|
| assignee設定 | × | × | × | ◎ | ○ | ○ |
| 実装（PR作成） | × | ◎ | × | ○ | × | ○ |
| PRレビュー | × | △ | ◎ | ○ | × | ○ |
| PRマージ | × | × | ◎ | ○ | × | ○ |

**管理系**:

| アクション | 起票者 | 開発者 | ﾚﾋﾞｭﾜー | Gﾘｰﾄﾞ | PM | Admin |
|---|---|---|---|---|---|---|
| phase advance | × | × | × | ◎ | ○ | ○ |
| phase skip | × | × | × | × | ◎ | ○ |
| stage advance/freeze | × | × | × | × | ◎ | ○ |
| ゲートIssueクローズ | × | × | × | △ | ◎ | ○ |
| 手戻り起票 | × | ◎ | ◎ | ◎ | ◎ | ○ |
| 手戻り承認 | × | × | × | △ | ◎ | ○ |
| group finalize | × | × | × | × | ◎ | ○ |
| member add/remove | × | × | × | × | ◎ | ○ |
| teraflow.yml変更 | × | × | × | × | ○ | ◎ |
| Secrets設定 | × | × | × | × | × | ◎ |
| changelog閲覧 | ○ | ○ | ○ | ○ | ○ | ◎ |

### 17.3 groups.yml でのメンバー・ロール定義

```yaml
# .github/groups.yml（members セクション）

roles:
  requester:
    display_name: "起票者"
  developer:
    display_name: "開発者"
  reviewer:
    display_name: "レビュワー"
  group_lead:
    display_name: "グループリード"
  pm:
    display_name: "プロジェクトマネージャー"
  admin:
    display_name: "管理者"

members:
  - github: "@biz-suzuki"
    roles:
      - role: requester

  - github: "@dev-a"
    roles:
      - role: developer
        group: frontend
      - role: reviewer
        group: backend

  - github: "@lead-b"
    roles:
      - role: group_lead
        group: backend
      - role: developer
        group: backend

  - github: "@pm-tanaka"
    roles:
      - role: pm
      - role: group_lead
        group: shared

  - github: "@admin-ops"
    roles:
      - role: admin
```

### 17.4 GitHub ネイティブ権限との対応

| teraflow ロール | GitHub リポジトリ権限 | GitHub Team |
|---|---|---|
| 起票者 | Read | `teraflow-requesters` |
| 開発者 | Write | `teraflow-{group}` |
| レビュワー | Write | `teraflow-{group}` |
| グループリード | Write | `teraflow-{group}` |
| PM | Maintain | `teraflow-pm` |
| Admin | Admin | `teraflow-admin` |

### 17.5 CODEOWNERS による PR マージ制御（ハードガード）

`teraflow setup codeowners` で `groups.yml` から自動生成する。

```
# .github/CODEOWNERS（自動生成）

docs/shared/                     @pm-tanaka @admin-ops
docs/backend/                    @lead-b @reviewer-backend
docs/frontend/                   @lead-f @reviewer-frontend
docs/infra/                      @lead-i @reviewer-infra

.github/teraflow.yml             @pm-tanaka @admin-ops
.github/project-state.yml        @pm-tanaka @admin-ops
.github/groups.yml               @pm-tanaka @admin-ops
.github/master-schedule.yml      @pm-tanaka @admin-ops
```

### 17.6 Actions によるソフトガード (`teraflow-permission-guard.yml`)

GitHub ネイティブでは制御できない権限を、Actions で「検知して差し戻す」方式で実装する。

**トリガー**: Issue クローズ時 / Issue ラベル変更時 / Issue コメント時（トリガーワード）

**処理**:
1. `groups.yml` からアクション実行者のロールを取得
2. アクション × ロールのマトリクスを照合
3. 権限不足の場合: アクションを差し戻し、理由をコメント、changelog に `permission.denied` を記録

**制御方式まとめ**:

| 制御内容 | 方式 | 強度 |
|---|---|---|
| PR マージ制限 | CODEOWNERS + Branch Protection | ハード |
| 必須レビュアー | CODEOWNERS | ハード |
| Issue クローズ制限 | Actions ソフトガード | ソフト |
| 「確定」権限制限 | Actions ソフトガード | ソフト |
| グループ別 Issue 操作制限 | Actions ソフトガード | ソフト |
| 管理ファイル変更制限 | CODEOWNERS | ハード |
| フェーズ進行・スキップ | CLI が groups.yml を照合して拒否 | ソフト |

### 17.7 CLI でのロール確認 (`teraflow role`)

```
$ teraflow role show @dev-a

  Member: @dev-a
  Roles:
    developer  (group: frontend)
    reviewer   (group: backend)

  Can do:
    ✅ Issue起票 (frontend)
    ✅ 実装・PR作成 (frontend)
    ✅ PRレビュー (backend)
    ❌ Issue起票 (backend) — developer権限はfrontendのみ
    ❌ 「確定」(frontend) — reviewer以上が必要
    ❌ phase advance — group_lead以上が必要
```

---

