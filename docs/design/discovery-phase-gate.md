---
node_id: "design:discovery-phase-gate"
title: "Discussion Discovery フェーズゲート設計書"
status: draft
created: "2026-04-14"
parent: "design:v-model-pipeline"
depends_on:
  - "design:requirements-discovery"
  - "design:rbac-gate-design"
---

# Discussion Discovery フェーズゲート設計書

## 1. 背景と課題

### 1.1 現状の非対称性

| ワークフロー | ゲート有無 | 挙動 |
|---|---|---|
| `teraflow-phase-gate.yml` (PR系) | ✅ あり | `project-state.yml` 読み込み → フェーズ/ステージ確認 |
| `teraflow-req-agent.yml` (Discussion系) | ❌ なし | 無条件で discovery 処理を実行 |

### 1.2 発生し得る問題

1. **未初期化リポ**: `teraflow init` 未実行のリポジトリで Discussion を作成すると、bot が応答してしまう
2. **フェーズ不一致**: `requirements` 以外のフェーズ（design/implement/test 等）でも discovery が動作する
3. **無制限セッション**: discovery セッション数の制限がない

### 1.3 重要な前提事実

調査の結果、タスク YAML の前提と異なる事実を確認した：

- **teraflow-check**: `.github/project-state.yml` が**既に存在**する（`phases.current: requirements`）
- **teraflow-canary**: リポジトリ自体が存在しない（ローカルにクローンなし）

→ E2E 互換性の懸念は当初想定より低い。ただし、将来的に `teraflow init` 未実行のリポでのテストや、フェーズが `requirements` 以外に進んだ状態でのテストに対応する仕組みは必要。

## 2. 設計案の比較

### 案A: ワークフロー冒頭でフェーズチェック（最小変更）

`teraflow-req-agent.yml` の早期ステップに `project-state.yml` 読み込み + フェーズ判定を追加。

```yaml
- name: Phase gate check
  id: phase_gate
  run: |
    if [ ! -f ".github/project-state.yml" ]; then
      echo "gate=no_project_state" >> $GITHUB_OUTPUT
      exit 0
    fi
    CURRENT_PHASE=$(python3 -c "
    import yaml
    s = yaml.safe_load(open('.github/project-state.yml')) or {}
    print(s.get('phases', {}).get('current', 'unknown'))
    " 2>/dev/null || echo "unknown")
    echo "phase=$CURRENT_PHASE" >> $GITHUB_OUTPUT
    if [ "$CURRENT_PHASE" != "requirements" ]; then
      echo "gate=phase_mismatch" >> $GITHUB_OUTPUT
    else
      echo "gate=pass" >> $GITHUB_OUTPUT
    fi
```

**利点:**
- 最小変更（ワークフロー内 1 ステップ追加 + 後続ステップの `if` 条件追加）
- 既存の RBAC チェック・API キーチェックと同じパターン
- Go コード変更不要 → リリースサイクルに依存しない
- `teraflow-phase-gate.yml` と同じ `project-state.yml` 読み取りロジックを使用

**欠点:**
- ワークフローテンプレート内ハードコード（`requirements` が文字列リテラル）
- 他のワークフローに再利用できない

### 案B: hook 条件タイプに `phases` を追加（設定ベース）

`HookConditionsCfg` に `Phases []string` フィールドを追加し、`matchesConditions()` でフェーズチェック。

```go
// config.go
type HookConditionsCfg struct {
    NotAuthor  []string `yaml:"not_author,omitempty"`
    Categories []string `yaml:"categories,omitempty"`
    Labels     []string `yaml:"labels,omitempty"`
    Paths      []string `yaml:"paths,omitempty"`
    Phases     []string `yaml:"phases,omitempty"`  // NEW
}

// types.go
type HookConditions struct {
    NotAuthor  []string
    Categories []string
    Labels     []string
    Paths      []string
    Phases     []string  // NEW
}

// HookContext に Phase フィールド追加
type HookContext struct {
    // ... existing fields ...
    Phase string  // NEW: current phase from project-state.yml
}
```

```go
// parser.go matchesConditions() に追加
if len(cond.Phases) > 0 && !contains(cond.Phases, ctx.Phase) {
    return false
}
```

```yaml
# teraflow.yml での設定例
hooks:
  on_discussion_created:
    - action: respond
      conditions:
        phases: [requirements]
  on_discussion_comment:
    - action: respond
      conditions:
        phases: [requirements]
```

**利点:**
- 設定ベースで汎用的（他の hook にも `phases` 条件を適用可能）
- 将来フェーズが増えても teraflow.yml の変更のみで対応
- テスト可能（Go ユニットテストで `matchesConditions` を検証）

**欠点:**
- Go 実装変更が必要（config.go, types.go, parser.go, parser_test.go）
- ワークフローテンプレート側でも `Phase` を `HookContext` に渡す処理が必要
- リリースサイクルに依存（teraflow バイナリの更新が必要）
- hook 条件チェックは `teraflow agent assign` 内部で実行されるため、ワークフローは既に Go インストール・teraflow ビルドまで進んでからフェーズ拒否される（無駄なランナー時間）

### 案C: A + B の組み合わせ（多重防御）

**利点:** 堅牢（ワークフロー層 + アプリケーション層の二重チェック）
**欠点:** 複雑性増加、メンテナンス対象が二箇所

## 3. 採用案の決定

### 推奨: **案A（ワークフロー冒頭フェーズチェック）**

**決定理由:**

| 評価軸 | 案A | 案B | 案C |
|--------|-----|-----|-----|
| 実装コスト | ◎ 小 | △ 中 | ✕ 大 |
| リリース速度 | ◎ テンプレ更新のみ | △ Go変更+リリース | ✕ 両方 |
| E2E影響 | ◎ 最小 | △ 中 | ✕ 大 |
| ランナー効率 | ◎ 早期終了 | △ Go install後に判定 | ○ 早期+後期 |
| 将来拡張性 | △ ハードコード | ◎ 設定ベース | ◎ 両方 |

**核心的判断:**
1. **ランナー効率**: 案B は Go インストール（~30秒）→ teraflow ビルド（~20秒）を経て初めてフェーズチェックに到達する。案A はチェックアウト直後（~2秒）に判定完了。GitHub Actions の課金は分単位なので、この差は無視できない。
2. **現実的なニーズ**: フェーズゲートの対象は現時点で discovery ワークフローのみ。汎用化の必要性は低い。
3. **段階的移行**: 将来 hook システムが成熟した時点で案B を追加実装し、案A を残すことで案C に自然移行可能。案A は案B の前提条件を壊さない。

## 4. 実装詳細

### 4.1 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `internal/actions/templates/teraflow-req-agent.yml` | フェーズゲートステップ追加 + 後続ステップの条件修正 |

Go コードの変更は不要。

### 4.2 ワークフロー変更内容

#### 4.2.1 新規ステップ: Phase Gate Check

`actions/checkout@v4` の直後、`React with EYES` の直後に挿入:

```yaml
      - name: Phase gate check
        id: phase_gate
        run: |
          # Check project-state.yml existence
          if [ ! -f ".github/project-state.yml" ]; then
            echo "gate=no_project_state" >> $GITHUB_OUTPUT
            echo "::notice::project-state.yml not found. teraflow init required."
            exit 0
          fi

          # Read current phase
          CURRENT_PHASE=$(python3 -c "
          import yaml
          s = yaml.safe_load(open('.github/project-state.yml')) or {}
          print(s.get('phases', {}).get('current', 'unknown'))
          " 2>/dev/null || echo "unknown")
          echo "phase=$CURRENT_PHASE" >> $GITHUB_OUTPUT

          if [ "$CURRENT_PHASE" = "requirements" ]; then
            echo "gate=pass" >> $GITHUB_OUTPUT
          else
            echo "gate=phase_mismatch" >> $GITHUB_OUTPUT
            echo "::notice::Current phase is '$CURRENT_PHASE', not 'requirements'. Discovery skipped."
          fi
```

#### 4.2.2 ゲート失敗時の案内コメント投稿

```yaml
      - name: Post phase gate message
        if: steps.phase_gate.outputs.gate != 'pass'
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          GATE="${{ steps.phase_gate.outputs.gate }}"
          PHASE="${{ steps.phase_gate.outputs.phase }}"

          if [ "$GATE" = "no_project_state" ]; then
            MSG="## ⚠️ プロジェクト未初期化

          このリポジトリはまだ teraflow で初期化されていません。
          要件探索（discovery）を利用するには、以下を実行してください:

          \`\`\`bash
          teraflow init
          \`\`\`

          > *teraflow phase gate による自動チェック*"
          else
            MSG="## ⚠️ フェーズ不一致

          現在のフェーズは \`$PHASE\` です。
          要件探索（discovery）は \`requirements\` フェーズでのみ利用可能です。

          現在のフェーズを確認するには:
          \`\`\`bash
          teraflow status
          \`\`\`

          > *teraflow phase gate による自動チェック*"
          fi

          # Post comment (reply to comment or top-level)
          if [ "${{ github.event_name }}" = "discussion_comment" ]; then
            REPLY_ARGS="-f replyToId=${{ github.event.comment.node_id }}"
          else
            REPLY_ARGS=""
          fi

          gh api graphql -f query='
            mutation($id: ID!, $body: String!, $replyToId: ID) {
              addDiscussionComment(input: {discussionId: $id, body: $body, replyToId: $replyToId}) {
                comment { id }
              }
            }' \
            -f id="${{ github.event.discussion.node_id }}" \
            -f body="$MSG" \
            $REPLY_ARGS
```

#### 4.2.3 後続ステップの条件修正

既存の各ステップの `if` 条件に `steps.phase_gate.outputs.gate == 'pass'` を **AND** で追加:

```yaml
# Before:
- name: Check required secrets
  id: check
  ...

# After:
- name: Check required secrets
  if: steps.phase_gate.outputs.gate == 'pass'
  id: check
  ...
```

影響を受けるステップ（`phase_gate.outputs.gate == 'pass'` を条件に追加）:

| ステップ名 | 現在の条件 | 変更後の条件 |
|---|---|---|
| Check required secrets | (なし) | `phase_gate == 'pass'` |
| Check RBAC permission | `check.has_key == 'true'` | `phase_gate == 'pass' && check.has_key == 'true'` |
| Post API key not set warning | `check.has_key == 'false'` | `phase_gate == 'pass' && check.has_key == 'false'` |
| Post permission denied comment | `rbac_check.allowed == 'False'` | `phase_gate == 'pass' && rbac_check.allowed == 'False'` |
| 以降の全ステップ | 各種条件 | `phase_gate == 'pass' && ...` |

**重要:** `React with EYES` ステップはゲート**前**に実行する。ユーザーに「受け付けた」ことを示すリアクションは、ゲートの結果に関わらず付ける。

### 4.3 ステップ順序（変更後）

```
1. checkout
2. React with EYES              ← ゲート前（常に実行）
3. Phase gate check             ← NEW
4. Post phase gate message      ← NEW（ゲート NG 時のみ）
5. Check required secrets       ← 条件追加: gate == 'pass'
6. Check RBAC permission        ← 条件追加
7. Post API key not set warning ← 条件追加
8. Post permission denied       ← 条件追加
9. setup-go                     ← 条件追加
10. Install teraflow            ← 条件追加
11. ... (以降すべて条件追加)
```

## 5. E2E テスト互換性

### 5.1 現状確認

| リポ | project-state.yml | phases.current | E2E影響 |
|------|-------------------|----------------|---------|
| teraflow-check | ✅ 存在 | `requirements` | **影響なし** — ゲート通過 |
| teraflow-canary | 未確認（ローカル未クローン） | — | 要確認 |

### 5.2 E2E で phases.current != "requirements" をテストしたい場合

E2E テストスクリプト内で `project-state.yml` を書き換えるステップを追加:

```bash
# E2E: フェーズゲート拒否テスト
python3 -c "
import yaml
with open('.github/project-state.yml') as f:
    state = yaml.safe_load(f)
state['phases']['current'] = 'design'
with open('.github/project-state.yml', 'w') as f:
    yaml.dump(state, f)
" 
# → Discussion 作成 → bot が「フェーズ不一致」コメントを返すことを検証
```

### 5.3 project-state.yml 未存在のテスト

```bash
# E2E: 未初期化リポテスト
mv .github/project-state.yml .github/project-state.yml.bak
# → Discussion 作成 → bot が「teraflow init してください」コメントを返すことを検証
mv .github/project-state.yml.bak .github/project-state.yml
```

## 6. サブタスク分解

| # | タスク | 依存 | 担当 | 見積 |
|---|--------|------|------|------|
| 1 | `teraflow-req-agent.yml` テンプレートにフェーズゲートステップ追加 | なし | 足軽 | 小 |
| 2 | 後続ステップの `if` 条件に `phase_gate == 'pass'` を追加 | #1 | #1と同一タスク | 小 |
| 3 | `setup actions --force` で生成 → teraflow-check に deploy → E2E 実行で検証 | #1,#2 | 家老/足軽 | 中 |
| 4 | フェーズゲート拒否ケースの E2E テスト追加（Phase D に新バリデーション） | #3 | 足軽 | 小 |

**#1 と #2 は同一 PR で実施可能。** テンプレート変更 1 ファイルのみ。

### 依存関係

```
#1 + #2 (テンプレート変更)
   ↓
#3 (deploy + E2E 検証)
   ↓
#4 (ゲート拒否テスト追加)
```

## 7. 実装後の検証方法

### 7.1 ローカル検証

```bash
# テンプレート差分確認
diff internal/actions/templates/teraflow-req-agent.yml.orig \
     internal/actions/templates/teraflow-req-agent.yml

# YAML lint
python3 -c "import yaml; yaml.safe_load(open('internal/actions/templates/teraflow-req-agent.yml'))"
```

### 7.2 E2E 検証（teraflow-check）

1. `teraflow setup actions --force` で teraflow-check にワークフローを deploy
2. 通常の discovery E2E を実行 → **PASS**（project-state.yml に `requirements` あり）
3. `project-state.yml` の `phases.current` を `design` に変更 → Discussion 作成 → **ゲート拒否コメント確認**
4. `project-state.yml` を削除 → Discussion 作成 → **未初期化コメント確認**
5. 元に戻す

### 7.3 確認チェックリスト

- [ ] `phases.current == "requirements"` → discovery 正常動作
- [ ] `phases.current != "requirements"` → フェーズ不一致コメント投稿、処理終了
- [ ] `project-state.yml` 未存在 → 未初期化コメント投稿、処理終了
- [ ] EYES リアクションはゲート結果に関わらず付与される
- [ ] 既存 E2E テスト（teraflow-check）が壊れない

## 8. 将来拡張（案B への移行パス）

案A が安定稼働した後、hook システムの成熟度に応じて案B を追加実装可能:

1. `HookConditionsCfg` に `Phases` フィールド追加
2. `matchesConditions()` にフェーズ判定追加
3. ワークフロー内で `teraflow agent assign` に `--phase` オプションを渡す
4. 案A のワークフロー内チェックはそのまま残す → 案C（多重防御）に自然移行

この移行は破壊的変更を含まないため、マイナーバージョンで実施可能。
