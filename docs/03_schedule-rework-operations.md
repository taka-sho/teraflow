## 11. マスタースケジュールと完了予測

### 11.1 スケジュール定義

```yaml
# .github/master-schedule.yml

project_name: ""

# 初期開発ステージのスケジュール
stages:
  初期開発:
    phases:
      - id: requirements
        name: 要求整理
        planned_start: null
        planned_end: null
        planned_days: null
        planned_artifacts: null   # 想定される成果物ファイル数
        actual_start: null
        actual_end: null
        actual_artifacts: null    # 実際に確定した成果物ファイル数
        artifact_path: "docs/01_requirements/REQ-*.md"
        gate_issue: null
      # 以下 要件定義, 基本設計, 詳細設計, 実装, テスト も同構造

  移行・リリース:
    phases:
      - id: migration-plan
        name: 移行計画
        planned_start: null
        planned_end: null
        planned_days: null
        # 以下同構造

  # 継続的改善はサイクルごとにスケジュールを持つ
  継続的改善:
    cycles:
      - id: "v1.1"
        name: "注文検索機能追加"
        planned_start: null
        planned_end: null
        phases: [...]
```

### 11.2 完了予測アルゴリズム

複合スコア法を適用する。進捗指標は **成果物ファイルの確定数** を基準とする。

```
見込み日数 = ベース日数 × 遅延倍率 × 膨張率 + 手戻りバッファ

  ベース日数     = master-schedule.yml の planned_days
  遅延倍率       = 直近完了フェーズの (actual_days / planned_days)
  膨張率         = 直近完了フェーズの (actual_artifacts / planned_artifacts)
  手戻りバッファ = 過去の手戻り平均解決日数 × 予測手戻り件数

消化速度 = 確定済み成果物ファイル数 / 経過日数
```

3点見積もり（楽観/標準/悲観）を算出する。継続的改善ステージでは、過去のサイクルの実績データを用いて次サイクルの予測精度を向上させる。

---

## 12. 手戻り管理とCoDD連携

### 12.1 手戻りの基本方針

手戻りが発生した場合、**フェーズ自体を正式に戻す**。個別Issueのラベルだけを変更するのではなく、`project-state.yml` のフェーズ状態を更新し、`phase_history` に戻った記録を残す。これにより、フェーズ状態と実態が常に一致する。

### 12.2 手戻りフロー

```
現在: 基本設計フェーズ
       │
       ▼  手戻りIssue #42 起票（rework テンプレート）
       │
  ┌────┴──────────────────────────────────────┐
  │  teraflow-rework-impact.yml が自動起動      │
  │                                            │
  │  1. Issue本文から関連要件ID・設計書を抽出    │
  │  2. codd impact で影響分析                  │
  │  3. 影響レポートをIssueコメントに投稿        │
  │  4. rework-log.yml に記録を追記             │
  │  5. PM/POにレビュー依頼                     │
  └────┬──────────────────────────────────────┘
       │
       ▼  PM/POが影響レポートを確認し判断
       │  手戻りIssueに「手戻り承認」とコメント
       │
  ┌────┴──────────────────────────────────────┐
  │  teraflow-rework-approve.yml が自動起動     │
  │                                            │
  │  1. project-state.yml を更新するPRを作成    │
  │     - 現フェーズの result を "reverted" に  │
  │     - reverted_to, rework_issue を記録     │
  │     - 戻し先フェーズを新規エントリとして追加 │
  │       (triggered_by: "rework")             │
  │     - current_phase を戻し先に変更          │
  │  2. PRレビュー・マージでフェーズが正式に遷移 │
  │  3. 関連するIssueのフェーズラベルを一括更新  │
  │  4. ダッシュボードに反映                    │
  └────┬──────────────────────────────────────┘
       │
       ▼  戻し先フェーズで修正作業
       │
       ▼  修正完了後、teraflow phase advance で再前進
          （通常のゲート条件チェックを経て）
```

### 12.3 影響分析レポート

手戻りIssue起票時に `codd impact` を実行し、以下のフォーマットでIssueコメントに投稿する。

```markdown
## 📊 影響分析レポート

**変更対象**: REQ-0012（注文検索の条件定義）
**発覚フェーズ**: 基本設計 → **戻し先**: 要件定義

### Green Band（高信頼度・自動修正可能）
| 対象成果物 | 深度 | 信頼度 |
|---|---|---|
| design:order-history-api | 1 | 0.90 |
| detail:order-service | 2 | 0.90 |

### Amber Band（要レビュー）
| 対象成果物 | 深度 | 信頼度 |
|---|---|---|
| design:db-design | 1 | 0.90 |
| test:test-strategy | 2 | 0.90 |

### Gray Band（参考情報）
| 対象成果物 | 深度 | 信頼度 |
|---|---|---|
| plan:implementation | 2 | 0.00 |

**影響スコア**: 4/12 成果物 (33%)

⚠️ PM/POが内容を確認し、手戻りを承認する場合は **手戻り承認** とコメントしてください。
```

### 12.4 手戻り後の再実行ポリシー

手戻りでフェーズを戻した場合、戻し先から再度フェーズを進める必要がある。どの範囲まで再実行するかは `re_execution_policy` で設定する。

```yaml
# .github/teraflow.yml（rework セクション）

rework:
  re_execution_policy: "impact_based"
  # full:         戻し先から全フェーズを再通過（全ゲート条件を再チェック）
  # impact_based: CoDD影響分析で影響ありの成果物のみ再確認
  # manual:       PM/POが都度判断
```

**`impact_based` の場合の動作**:

```
基本設計で手戻り → 要件定義に戻る

  要件定義フェーズ: 該当要件のみ修正 → ゲートチェック → advance
  基本設計フェーズ: Green対象の設計書のみ再レビュー → ゲートチェック → 続行
```

**`full` の場合の動作**:

```
基本設計で手戻り → 要件定義に戻る

  要件定義フェーズ: 全ゲート条件を再チェック → advance
  基本設計フェーズ: 全ゲート条件を再チェック → 続行
```

### 12.5 手戻り記録

```yaml
# docs/rework-log.yml（自動追記）
reworks:
  - id: RW-001
    date: "2026-05-20"
    detected_in: "基本設計"
    root_cause_phase: "要件定義"
    rework_issue: "#42"
    reason: "注文検索の条件が未定義"

    # 承認記録
    approved_by: "@pm-tanaka"
    approved_at: "2026-05-20"

    # フェーズ遷移記録
    phase_reverted_from: "基本設計"
    phase_reverted_to: "要件定義"
    phase_history_entries:         # phase_history に追記されたエントリ番号
      reverted_entry: 3            # result: "reverted" になったエントリ
      new_entry: 4                 # triggered_by: "rework" の新エントリ

    # CoDD 影響分析
    impact_score: "4/12"
    affected_artifacts:
      green: ["design:order-api", "detail:order-service"]
      amber: ["test:test-strategy"]
      gray: ["plan:implementation"]

    # 再実行ポリシー
    re_execution_policy: "impact_based"

    # 解決記録（手戻り起点のフェーズが再度 advance で通過した時に自動更新）
    resolution_date: null
    days_to_resolve: null
    phases_re_executed: []         # 再通過したフェーズのリスト
```

### 12.6 手戻り承認コマンド (`teraflow rework approve`)

CLI からも手戻り承認を実行できる。

```
$ teraflow rework approve --issue 42

Rework Issue: #42
  Detected in: 基本設計
  Revert to:   要件定義
  Impact:      4/12 成果物 (33%)

This will:
  1. Revert current phase from 基本設計 to 要件定義
  2. Record phase history (基本設計 → reverted)
  3. Add new phase entry (要件定義, triggered_by: rework)
  4. Update rework-log.yml
  5. Create PR to update project-state.yml

Confirm? (y/n)
  ✓ PR #55 created: "rework: フェーズ後退 基本設計→要件定義 (Issue #42)"
```

### 12.7 手戻り統計 (`teraflow rework stats`)

```
$ teraflow rework stats

  累計手戻り: 3回
  平均解決日数: 5.2日
  累計遅延影響: +14日

  原因フェーズ別:
    要求整理起因:  1回 (33%)
    要件定義起因:  2回 (67%)

  発覚フェーズ別:
    基本設計で発覚: 2回
    詳細設計で発覚: 1回

  再実行ポリシー別:
    impact_based: 2回
    full: 1回

  → 要件定義の精度向上が最優先改善ポイント
```

### 12.8 ダッシュボードへのフェーズ履歴反映

フェーズ履歴をタイムラインとして表示する。手戻りによる中断と再開が視覚的に把握できる。

```
─── フェーズ履歴 ──────────────────────────────────────────

  要求整理   04/01━━━04/18 ✅ completed
  要件定義   04/21━━━05/09 ✅ completed
  基本設計   05/12━━05/20 🔄 reverted (→要件定義, #42)
  要件定義   05/20━━━05/25 ✅ completed (rework #42)
  基本設計   05/26━━━━━━━ ◀ 進行中 (rework #42)
  詳細設計   ─────────────
  実装       ─────────────
  テスト     ─────────────

  🔄 手戻り累計: 1回
  📉 手戻りによる追加日数: +8日

─── スケジュール予測（手戻り反映済み） ─────────────────────

  フェーズ        予定      標準(手戻り前)   標準(現在)
  基本設計(2回目) 05/30     06/10           06/12
  詳細設計       06/20      07/14           07/20
  実装           07/25      08/28           09/02
  テスト         08/15      09/24           09/30
```

追加仕様: 継続的改善ステージでは、稼働中システムへの影響分析がより重要になる。`codd impact` の結果に「本番影響の有無」を追加表示する。

---

## 13. 運用ステージ: 障害管理

### 13.1 障害報告の自動処理 (`teraflow-incident-agent.yml`)

**トリガー**: `incident` ラベル付きIssue作成時

**処理フロー**:
1. Issue本文から事象・影響範囲を解析
2. 関連するコードベースを調査（Agent）
3. 原因候補と修正方針をIssueコメントに投稿
4. `docs/incident-log.yml` に障害記録を追記

### 13.2 障害記録

```yaml
# docs/incident-log.yml（自動追記）
incidents:
  - id: INC-001
    date: "2026-09-05T14:30:00+09:00"
    severity: "major"
    summary: "注文確定時にタイムアウトエラー"
    impact: "顧客の注文が完了しない"
    root_cause: null             # 調査完了時に記入
    resolution: null
    resolved_at: null
    mttr_hours: null             # 自動計算
    related_issues: ["#200"]
    postmortem: null             # ポストモーテムへのリンク
```

### 13.3 障害統計 (`teraflow incident stats`)

```
$ teraflow incident stats

  Period: 2026-08 〜 2026-10

  Total incidents: 8
  By severity:
    Critical: 1
    Major:    3
    Minor:    4

  MTTR (Mean Time To Resolve):
    Critical: 2.0h
    Major:    4.5h
    Minor:    8.2h

  Top root causes:
    1. データ不整合 (3件)
    2. 外部API障害 (2件)
    3. メモリリーク (1件)
```

---

## 14. 保守ステージ: 技術的負債管理

### 14.1 定期スコアリングとの連携

ハーネスエンジニアリングの定期スコアリング結果に基づき、スコアが閾値を下回った観点について保守Issueを自動起票する。

```
スコアリング実行 → セキュリティ: 45/100 (閾値: 60)
  → 自動起票: "セキュリティスコア改善: 依存ライブラリの脆弱性対応"
     ラベル: stage:保守, maintenance, type:セキュリティ, priority:高
```

### 14.2 保守作業エージェント (`teraflow-maintenance-agent.yml`)

**トリガー**: `agent:保守実行` ラベルの付与

**対応可能な作業種別**:
- ライブラリの依存関係更新（バージョンアップ）
- 非推奨APIの置き換え
- コーディング規約違反の修正
- 不要コードの検知・削除

**Agentに渡すコンテキスト**: CLAUDE.md + Golden Principles + 対象のIssue本文 + 関連するコード

---

## 15. 実装フェーズ: 人間/Agentハイブリッド
