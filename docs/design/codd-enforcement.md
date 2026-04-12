---
codd:
  node_id: "design:codd-enforcement"
  title: "要件変更→CoDD文書更新強制 設計書"
  depends_on:
    - id: "design:v-model-pipeline"
      relation: extends
    - id: "design:requirements-discovery"
      relation: extends
    - id: "design:rbac-gate-design"
      relation: extends
  status: "draft"
  review_required: "approve"
  created_at: "2026-04-12"
  updated_at: "2026-04-12"
---

# 要件変更→CoDD文書更新強制 設計書

> cmd_203 — 要件が変更されたら、実装の前に必ず CoDD 文書を更新させる仕組み


## 1. 課題と目的

### 1.1 課題

現状の teraflow では、要件変更後に CoDD 文書を更新せずに実装を進めることが可能である。これにより:

- 要件と設計書の乖離が蓄積する
- impact 分析の結果が反映されない（Amber/Gray バンドが放置される）
- V字モデルの対称性が崩れる（要件変更→受入テスト仕様の不整合）

### 1.2 目的

**要件変更→CoDD 文書更新→実装** の順序を強制し、文書と実装の一貫性を常に維持する。

### 1.3 原則

| 原則 | 内容 |
|------|------|
| **Docs First** | 実装 PR は、対応する CoDD 文書が最新であることを前提とする |
| **段階的強制** | Phase 1 は警告、Phase 2 でブロック、段階的に導入する |
| **自動化優先** | 人間に「文書を更新しろ」と言うだけでなく、更新 PR を自動生成する |
| **既存フロー統合** | 新しい仕組みではなく、既存の validate/impact/hook を拡張する |


## 2. フロー全体像

```
[要件変更検知]
  │
  ├─ (A) Discovery/grill-me で回答変更
  │   └─ on_requirement_changed フック発火
  │
  ├─ (B) 要件文書 PR のマージ（内容変更あり）
  │   └─ on_push フック + 変更検知
  │
  └─ (C) 手動: teraflow impact --node req:xxx
      └─ 開発者が明示的に実行
  │
  ▼
[影響範囲特定]
  │
  ├─ teraflow impact --node <changed-node>
  │   ├─ 下流ノード一覧 + バンド判定 (Green/Amber/Gray)
  │   └─ CoDD 文書の change_impact フィールド更新
  │
  ▼
[CoDD 文書更新]
  │
  ├─ (自動) skill: codd-update → 更新 PR 自動生成
  │   └─ Amber: 差分追記 PR / Gray: 全文再生成 PR
  │
  ├─ (手動) Issue 作成 "Update docs for <node-id>"
  │   └─ ラベル: docs-required
  │
  ▼
[ゲート検証]
  │
  ├─ 実装 PR 作成時: teraflow-phase-gate.yml 拡張
  │   ├─ depends_on ノードに Amber/Gray がないか検証
  │   ├─ 警告 (Phase 1) またはブロック (Phase 2)
  │   └─ "docs-required" ラベル自動付与
  │
  ▼
[文書更新完了 → 実装続行]
  │
  └─ CoDD 文書 PR マージ → change_impact が Green に戻る
      └─ 実装 PR のゲートが通過可能になる
```


## 3. 要件変更の検知

### 3.1 検知パターン

| パターン | トリガー | 検知方法 |
|---------|---------|---------|
| Discovery セッション中の回答変更 | grill-me で既存回答を修正 | discovery スキル内で回答変更を検出 → フック発火 |
| 要件文書 PR のマージ | `docs/requirements/*.md` の変更がマージ | on_push フック + パス条件 |
| 手動の impact 実行 | 開発者が `teraflow impact` を実行 | CLI 出力で Amber/Gray を通知 |

### 3.2 新規フックイベント: on_requirement_changed

既存の HookEvent に新しいイベントを追加する。

```go
// internal/hooks/types.go（拡張）
const (
    EventDiscussionCreated  HookEvent = "on_discussion_created"
    EventDiscussionComment  HookEvent = "on_discussion_comment"
    EventConfirmation       HookEvent = "on_confirmation"
    EventPush               HookEvent = "on_push"
    EventPROpened           HookEvent = "on_pr_opened"
    EventRequirementChanged HookEvent = "on_requirement_changed"  // 新規
)
```

### 3.3 teraflow.yml への設定追加

```yaml
# .github/teraflow.yml
hooks:
  on_requirement_changed:
    - action: impact_analysis
      conditions:
        paths: ["docs/requirements/*.md"]
    - action: codd_update
      skill: codd-update
      conditions:
        paths: ["docs/requirements/*.md"]

  on_push:
    - action: index_update
    - action: detect_requirement_change    # 新規: 要件ファイル変更を検知
      conditions:
        paths: ["docs/requirements/*.md", ".teraflow/discovery/*.yaml"]
```


## 4. 影響範囲特定

### 4.1 impact 分析の拡張

既存の `teraflow impact` コマンドに CoDD 文書更新の必要性を判定するロジックを追加する。

```go
// internal/pipeline/impact.go（拡張）
type ImpactResult struct {
    ChangedNode    string
    AffectedNodes  []AffectedNode
    RegenRequired  []string         // Gray: 再生成必要
    ReviewNeeded   []string         // Amber: レビュー必要
    DocsRequired   []string         // 新規: 文書更新が必要なノード
    Summary        map[string]int
    // ...
}
```

`DocsRequired` は `RegenRequired`（Gray）と `ReviewNeeded`（Amber）の和集合から、設計文書ノード（`design:*`, `detail:*`）のみを抽出したもの。

### 4.2 change_impact フィールドの自動更新

impact 分析実行時に、影響を受けるノードの CoDD frontmatter `change_impact` フィールドを自動更新する。

```yaml
---
codd:
  node_id: "design:auth-system"
  change_impact: "amber"              # impact 分析で自動更新
  change_impact_source: "req:feature-auth"  # 変更元ノード
  change_impact_at: "2026-04-12"      # 判定日時
---
```

これにより、`teraflow validate` が `change_impact: amber|gray` のノードを検出できる。


## 5. 強制メカニズム

### 5.1 CI ゲート拡張（teraflow-phase-gate.yml）

既存の phase-gate ワークフローに CoDD 文書整合性チェックを追加する。

```yaml
# teraflow-phase-gate.yml（拡張部分）
- name: Check CoDD document freshness
  id: codd-check
  run: |
    # 実装対象モジュールの depends_on を辿り、
    # Amber/Gray の CoDD 文書がないか検証
    RESULT=$(teraflow validate --level 3 --format json)
    STALE=$(echo "$RESULT" | jq '[.errors[] | select(.error_type == "stale_dependency")] | length')
    
    if [ "$STALE" -gt 0 ]; then
      echo "::warning::CoDD文書に未更新の依存があります（${STALE}件）"
      echo "stale_count=$STALE" >> "$GITHUB_OUTPUT"
    fi

- name: Add docs-required label
  if: steps.codd-check.outputs.stale_count > 0
  run: |
    gh pr edit "$PR_NUMBER" --add-label "docs-required"
    gh pr comment "$PR_NUMBER" --body "$(cat <<'COMMENT'
    ## ⚠️ CoDD 文書の更新が必要です

    この PR が依存する CoDD 文書に未反映の要件変更があります。
    実装の前に、以下の文書を更新してください。

    $(teraflow impact --node "$CHANGED_NODE" --output text | grep -E 'amber|gray')

    **対応方法:**
    1. `teraflow impact --apply impact.json` で更新 Issue を作成
    2. 文書を更新して PR を作成・マージ
    3. この PR を再度チェック

    > `docs-required` ラベルが付いている間、マージはブロックされます。
    COMMENT
    )"
```

### 5.2 validate コマンドの拡張

`teraflow validate` に「要件変更後の文書未更新」を検出する検証ルールを追加する。

```go
// internal/validate/validator.go（拡張）
type ValidationError struct {
    NodeID    string
    ErrorType string    // 既存 + "stale_dependency" 追加
    Level     int
    Message   string
    Severity  string
}

// Level 3 に追加: stale_dependency チェック
func (v *Validator) checkStaleDependencies(nodeID string) []ValidationError {
    entry := v.index.FindByNodeID(nodeID)
    if entry == nil {
        return nil
    }
    var errors []ValidationError
    for _, dep := range entry.DependsOn {
        depEntry := v.index.FindByNodeID(dep)
        if depEntry == nil {
            continue
        }
        // frontmatter の change_impact を読み取り
        impact := v.readChangeImpact(depEntry.Path)
        if impact == "amber" || impact == "gray" {
            errors = append(errors, ValidationError{
                NodeID:    nodeID,
                ErrorType: "stale_dependency",
                Level:     3,
                Message:   fmt.Sprintf("depends on %s which has change_impact=%s (update required)", dep, impact),
                Severity:  "error",
            })
        }
    }
    return errors
}
```

### 5.3 ラベルによる可視化

| ラベル | 用途 | 付与タイミング |
|--------|------|---------------|
| `docs-required` | CoDD 文書更新待ち | CI ゲートで Amber/Gray 検出時 |
| `docs-updated` | 文書更新完了 | CoDD 文書 PR マージ後 |
| `impact` | 影響分析済み | `teraflow impact --apply` 実行後 |

### 5.4 強制レベル設定

teraflow.yml で強制レベルを設定可能にする。

```yaml
# .github/teraflow.yml
enforcement:
  codd_freshness: "warn"    # warn | block（デフォルト: warn）
  # warn: CI コメントで警告するが、マージは許可
  # block: docs-required ラベルが付いている間、マージをブロック
```


## 6. CoDD 文書更新スキル

### 6.1 新規スキル: codd-update

要件変更に対応する CoDD 文書を自動更新するスキルを追加する。

```yaml
# skills/codd-update.yml
name: codd-update
version: "1"
description: "要件変更に伴うCoDD文書の自動更新"

trigger:
  labels: ["docs-required", "codd-update"]
  agent_types: ["implement"]

prompts:
  system: |
    あなたはCoDD文書更新のスペシャリストです。
    要件変更の影響を受けたCoDD文書を正確に更新します。

    ## ルール
    - 変更元の要件文書と変更内容を正確に把握する
    - 影響を受ける文書のうち、変更が必要な箇所のみを修正する
    - frontmatter の change_impact を "green" に戻す
    - depends_on の整合性を維持する
    - 変更理由を commit メッセージに含める

  update_amber: |
    以下のCoDD文書がAmber（要確認）と判定されました。
    変更元の要件と比較し、追記・修正が必要な箇所を特定して更新してください。
    小規模な追従修正を想定しています。

  update_gray: |
    以下のCoDD文書がGray（要再生成）と判定されました。
    変更元の要件に基づき、文書を再生成してください。
    既存の構造を維持しつつ、内容を全面的に見直してください。

context:
  include:
    - "docs/requirements/*.md"
    - "docs/design/*.md"
  max_context_tokens: 8000

options:
  max_tokens: 8192
  temperature: 0.2
```

### 6.2 更新 PR の自動生成

```
[on_requirement_changed フック発火]
  │
  ├─ teraflow impact --node <changed-req> --output json > /tmp/impact.json
  │
  ├─ Amber ノード → codd-update スキル (update_amber プロンプト)
  │   └─ 差分追記 PR 作成 (review_required: review)
  │
  └─ Gray ノード → codd-update スキル (update_gray プロンプト)
      └─ 全文再生成 PR 作成 (review_required: approve)
```

### 6.3 ワークフロー: teraflow-codd-update.yml

```yaml
name: CoDD Update
on:
  push:
    paths: ["docs/requirements/*.md"]
    branches: [main]

jobs:
  detect-and-update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install teraflow
        run: go install ./...

      - name: Detect changed requirement nodes
        id: detect
        run: |
          CHANGED=$(git diff --name-only HEAD~1 HEAD -- docs/requirements/ | \
            xargs -I{} teraflow scan --file {} --format node_id)
          echo "nodes=$CHANGED" >> "$GITHUB_OUTPUT"

      - name: Run impact analysis
        if: steps.detect.outputs.nodes != ''
        run: |
          for NODE in ${{ steps.detect.outputs.nodes }}; do
            teraflow impact --node "$NODE" --output json >> impact-results.json
          done

      - name: Create update tasks
        if: steps.detect.outputs.nodes != ''
        run: |
          # Amber/Gray ノードに対して更新 Issue を作成
          teraflow impact --apply impact-results.json

      - name: Auto-update documents (Amber)
        if: steps.detect.outputs.nodes != ''
        run: |
          # codd-update スキルで Amber 文書を自動更新
          AMBER=$(jq -r '.[] | .review_needed[]' impact-results.json 2>/dev/null || true)
          for NODE in $AMBER; do
            teraflow agent assign --type implement --skill codd-update \
              --input "update_amber:$NODE" --create-pr
          done
```


## 7. Discovery との統合

### 7.1 回答変更の検知

Discovery セッション中にユーザーが既存の回答を変更した場合、discovery スキルが変更を検知して `on_requirement_changed` フックを発火する。

```go
// internal/discovery/session.go（拡張）
func (s *SessionState) UpdateAnswer(branchID, newAnswer string) (changed bool) {
    branch := s.FindBranch(branchID)
    if branch == nil {
        return false
    }
    if branch.Answer == newAnswer {
        return false
    }
    oldAnswer := branch.Answer
    branch.Answer = newAnswer
    branch.Status = StatusAnswered
    
    // 変更履歴を記録
    s.ChangeLog = append(s.ChangeLog, ChangeEntry{
        BranchID:  branchID,
        OldAnswer: oldAnswer,
        NewAnswer: newAnswer,
        Timestamp: time.Now(),
    })
    return true
}
```

### 7.2 「要求確定」後の変更

要求確定（`on_confirmation`）後に要件文書が変更された場合:

1. `on_push` フックで要件ファイルの変更を検知
2. `teraflow impact` で下流への影響を分析
3. `codd-update` スキルで設計文書の更新 PR を自動生成
4. 実装 PR には `docs-required` ラベルが付与される


## 8. teraflow 自身への適用（ドッグフーディング）

### 8.1 前提

cmd_202 で `.codd/codd.yaml` が修正され、teraflow 自身の CoDD 文書管理が正常化されている前提。

### 8.2 適用手順

1. `.github/teraflow.yml` に enforcement 設定を追加:
   ```yaml
   enforcement:
     codd_freshness: "warn"  # まず warn で開始
   ```

2. `teraflow validate --full` を実行し、既存の stale_dependency を洗い出す

3. 既存の未更新文書を一括更新する（初期コスト）

4. CI で phase-gate が動作していることを確認

5. 問題なければ `codd_freshness: "block"` に昇格

### 8.3 期待される効果

- 要件変更（Discussion での仕様変更）が確実に設計書に反映される
- `teraflow validate --full` が常に Green を維持できる
- V字モデルの左側（設計）と右側（検証）の整合性が自動的に保証される


## 9. validate/impact コマンドの変更詳細

### 9.1 validate の変更

| 変更内容 | 対象レベル | 詳細 |
|---------|-----------|------|
| `stale_dependency` エラー追加 | Level 3 | depends_on ノードに change_impact=amber/gray があればエラー |
| `change_impact` フィールド読み取り | Level 3 | CoDD frontmatter から change_impact を解析 |

既存の Level 1-2 チェックには影響なし。`--level 2` 以下で実行すれば従来通りの動作。

### 9.2 impact の変更

| 変更内容 | 詳細 |
|---------|------|
| `DocsRequired` フィールド追加 | 設計文書ノード（design:*, detail:*）のうち Amber/Gray のものを列挙 |
| `--update-frontmatter` フラグ追加 | 影響ノードの change_impact フィールドを自動更新（デフォルト: false） |
| `--apply` 拡張 | docs-required ラベル付き Issue も作成 |

```bash
# 影響分析 + frontmatter 自動更新
teraflow impact --node req:feature-auth --update-frontmatter

# 影響分析 + Issue 作成 + docs-required ラベル付与
teraflow impact --node req:feature-auth --output json > impact.json
teraflow impact --apply impact.json
```


## 10. 段階的導入計画

### Phase 1: CI ゲート警告（S サイズ）

**目標**: 文書未更新を検知して警告する（マージはブロックしない）

| サブタスク | 内容 | 工数 |
|-----------|------|------|
| 1-1 | validate に `stale_dependency` エラー追加 | S |
| 1-2 | CoDD frontmatter の `change_impact` 読み取り実装 | S |
| 1-3 | teraflow-phase-gate.yml に警告ステップ追加 | S |
| 1-4 | `docs-required` ラベル自動付与 | S |
| 1-5 | teraflow.yml に `enforcement.codd_freshness` 設定追加 | S |
| 1-6 | テスト | S |

**ゲート条件**: `change_impact: amber` のノードがある状態で実装 PR を作成すると、PR コメントに警告が表示されること

### Phase 2: codd-update スキル実装（M サイズ）

**目標**: 文書更新 PR の自動生成

| サブタスク | 内容 | 工数 |
|-----------|------|------|
| 2-1 | `skills/codd-update.yml` 作成 | S |
| 2-2 | `on_requirement_changed` フックイベント追加 | S |
| 2-3 | impact コマンドに `--update-frontmatter` フラグ追加 | S |
| 2-4 | impact コマンドの `DocsRequired` フィールド実装 | S |
| 2-5 | teraflow-codd-update.yml ワークフロー作成 | M |
| 2-6 | Discovery セッション内の回答変更検知 | M |
| 2-7 | テスト | M |

**ゲート条件**: 要件文書がマージされたら、Amber/Gray の設計文書に対して更新 PR が自動生成されること

### Phase 3: 完全自動化 + ブロック（M サイズ）

**目標**: 文書未更新時にマージをブロック

| サブタスク | 内容 | 工数 |
|-----------|------|------|
| 3-1 | `enforcement.codd_freshness: "block"` 対応 | S |
| 3-2 | `docs-required` ラベル除去の自動化（文書 PR マージ時） | S |
| 3-3 | ブロック解除フロー（`docs-updated` ラベル付与→ゲート再実行） | M |
| 3-4 | Gray ノードの全文再生成フロー | M |
| 3-5 | teraflow 自身でのドッグフーディング | M |
| 3-6 | E2E テスト | M |

**ゲート条件**: `codd_freshness: "block"` 設定時に、`docs-required` ラベルがある PR がマージできないこと


## 11. リスクと緩和策

| リスク | 影響 | 緩和策 |
|--------|------|--------|
| 過剰な文書更新要求 | 軽微な要件修正でも全下流が Amber になる | impact の `amber_threshold` で影響度閾値を調整。Green 判定を広めにする |
| 自動更新 PR の品質 | AI が不適切な更新を生成する可能性 | Amber は review、Gray は approve で人間レビューを強制 |
| 既存プロジェクトへの導入負荷 | 初回 validate で大量の stale_dependency が検出される | Phase 1 は warn のみ。段階的に対応 |
| CI 実行時間の増加 | validate Level 3 の追加チェック | change_impact フィールドの読み取りは軽量（frontmatter のみ） |
| フック発火の誤検知 | requirements ディレクトリ外の変更で発火 | paths 条件で厳密にフィルタリング |
| ブロックによる開発速度低下 | 文書更新完了まで実装 PR がマージできない | 緊急時は `enforcement.codd_freshness: "warn"` に一時変更可能 |
