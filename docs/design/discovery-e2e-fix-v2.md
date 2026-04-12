---
codd:
  node_id: "design:discovery-e2e-fix-v2"
  title: "Discovery v0.5.17 E2E残存問題の根本原因分析と修正設計"
  depends_on:
    - id: "design:requirements-discovery"
      relation: extends
    - id: "design:discovery-multiround"
      relation: extends
    - id: "design:discovery-e2e-fix"
      relation: extends
  tags:
    - discovery
    - e2e-fix
    - deployment
    - multiround
  status: draft
---

# Discovery v0.5.17 E2E残存問題の根本原因分析と修正設計

## 1. v0.5.17 E2Eテスト結果サマリ

| Phase | 結果 | 詳細 |
|-------|------|------|
| A (setup) | PASS | clone, version update, push 全て正常 |
| B (initial) | PASS | 質問3問生成OK、V1-V5 all pass |
| C (rounds) | **FAIL** | V6(new_comment), V7(confirmed_count), V11(round_count) fail |
| D (confirm) | **Partial** | V12,V16,V18 pass / V13,V14,V15,V17 fail |
| E (cleanup) | PASS | |

## 2. 問題1（最重要）: discussion_comment後にconfirmedが更新されない

### 2.1 エビデンス

**Discussion #70 のコメント構造（GraphQL replies含む調査結果）:**

```text
[TOP] github-actions 16:32:12 — 初回応答（質問3問: 対象ユーザー/最優先範囲/成功条件）
  [REPLY] taka-sho 16:32:27 — "1a 2b 3a"
  [REPLY] github-actions 16:33:19 — bot応答（再度同じ質問3問を提示）
  [REPLY] taka-sho 16:33:37 — "要求確定"
[TOP] github-actions 16:34:47 — 📋 要件確定（CoDD ドラフト出力）
```

**重要発見:** botは「1a 2b 3a」に対してリプライを返している（16:33:19）。ただしリプライは**ネストされたreply**であり、E2Eスクリプトのtop-levelクエリでは取得できなかった。

**summary.yaml:**
```yaml
confirmed: []        # ← 空のまま
round_count: 1       # ← 増加している（ワークフロー自体は実行された）
```

**state.yaml:** confirmノード1つのみ。round 1のノードはconfirmにより上書き。

### 2.2 根本原因: パースロジック未デプロイ

| ファイル | 行数 | パースロジック | 状態 |
|---------|------|-------------|------|
| テンプレート（teraflow本体） | **1492行** | `parse_user_answer`, `extract_questions_from_comment`, `stage1_parse_user_answers` 等あり | 最新 |
| teraflow-check実稼働版 | **1198行** | **なし** | 旧バージョン |

**差分: 約300行（294行追加）。** cmd_216 で設計したパースロジック全体が含まれる。

```text
テンプレート1492行版の追加内容:
- discovery_fallback.py（フォールバック質問JSON）
- infer_category() — 質問カテゴリ推定
- extract_questions_from_comment() — AIコメントから質問抽出
- build_unparsed_portion() — パース残部分記録
- parse_user_answer() — "1a 2b 3a" パース
- find_recent_assistant_comment() — 直前AI応答取得
- find_question_node() — 質問ノードマッチング
- stage1_parse_user_answers() — Stage 1 パース統合
- mark_resolved_from_parse() — パース結果のtree適用
- simplify_question_label() — 質問ラベル簡略化
- update_confirmed_from_parse() — confirmed更新
- merge_ai_resolved() — AI解析結果マージ
```

**因果連鎖:**
```text
1492行版が teraflow-check にデプロイされていない
  → 1198行版が稼働中（パースロジックなし）
    → ユーザー「1a 2b 3a」がパースされない
      → resolved_branches 空
        → confirmed 空のまま
          → bot が同じ質問を再質問
```

### 2.3 なぜデプロイされていないか

`teraflow setup actions --force` で 1492行版テンプレートを teraflow-check に展開しようとすると **workflow file issue** が発生する（問題3で詳述）。そのため 1198行版のまま運用されている。

### 2.4 副次的問題: E2Eスクリプトの検証不備

E2Eスクリプトの `V6_new_comment` 検証が **top-level comments のみ**をチェックしており、ネストされたrepliesを検出できない。

```text
現行: comments(first:20) → top-level のみ
必要: comments(first:20) { nodes { replies(first:10) { nodes { ... } } } }
```

botは実際にはreplyを返しているのに、V6が FAIL と判定した。

## 3. 問題2: CoDD document構造不備

### 3.1 エビデンス

```text
V14(codd_frontmatter): FAIL
V15(section_structure): FAIL
V17(pr_frontmatter): FAIL
```

ただし V16(pr_created): PASS、V18(discussion_ref): PASS。つまりPR#71は作成されたが、CoDD構造が不備。

### 3.2 根本原因

**問題1の直接的帰結。** confirmed=[] の状態で `teraflow doc generate` に進むため:
- ドラフトファイルに有意な要件データがない
- CoDD frontmatter が生成されない or 不完全
- セクション構造（スコープ/機能要件等）が空

### 3.3 修正方針

**問題1（パースロジックのデプロイ）が解決すれば自動解消する見込み。** confirmed が蓄積された状態で doc generate が呼ばれれば、有意なCoDD文書が生成される。

ただし、confirmed=[] の場合のガード（cmd_218 Fix 3-Bで設計済み）は引き続き必要:
- confirmed 空時は doc generate をスキップし警告コメントを返す
- これは 1492行版テンプレートへの追加修正として実装

## 4. 問題3: setup actions --force 生成workflowの不具合

### 4.1 エビデンス

テンプレート（1492行版）をteraflow-checkにデプロイしようとすると「workflow file issue」が発生。

### 4.2 差分分析

1198行版（動作中）と1492行版（問題あり）の主要差分:

| 差分箇所 | 内容 | 行数 |
|---------|------|------|
| フォールバックPython（PYEOF heredoc） | `cat > /tmp/discovery_fallback.py << 'PYEOF'` | +40行 |
| `extract_questions_from_comment` | AIコメントから質問抽出関数 | +30行 |
| `build_unparsed_portion` | パース残部分記録 | +25行 |
| `parse_user_answer` | ユーザー回答パース | +40行 |
| `find_question_node` | 質問ノードマッチング | +17行 |
| `stage1_parse_user_answers` + `mark_resolved_from_parse` | Stage 1 パース統合 | +65行 |
| `update_confirmed_from_parse` + `merge_ai_resolved` | confirmed更新 | +40行 |
| `doc context` ステップ | 既存文書インデックス参照 | +70行 |
| その他（simplify_question_label, infer_category等） | ユーティリティ | +20行 |

**PYEOF heredoc のインデント:**

```text
テンプレート内の PYEOF 位置:
  行545: "          cat > /tmp/discovery_fallback.py << 'PYEOF'"  (column 10)
  行583: "          PYEOF"                                         (column 10)
```

GitHub Actions の `run: |` ブロックでは、YAML パーサーがベースインデントを除去する。`run: |` の内容がcolumn 10で始まるなら、column 10が column 0として解釈される。つまりPYEOFのインデントは正しい。

### 4.3 推定原因

#### 候補A: テンプレート変数未置換

テンプレートには `{{.TeraflowVersion}}` が含まれる（174行目）。`setup actions` がテンプレート変数を正しく置換しないと、`{{` がYAMLとして不正になる可能性がある。

```yaml
# テンプレート
go install "github.com/taka-sho/teraflow@{{.TeraflowVersion}}"

# 1198行版（正しく置換済み）
go install "github.com/taka-sho/teraflow@v0.5.17"
```

→ この差分は1箇所のみ。`setup actions` がこの変数を正しく置換しているかが鍵。

#### 候補B: YAML heredoc内のエスケープ問題

1492行版で追加されたPython コード内に、YAML が誤解釈する文字列が含まれている可能性:
- `\n` （JSON文字列内の改行）→ YAMLパーサーが解釈を試みる
- `{` `}` （JSON/Python辞書リテラル）→ YAML のフロー記法と競合する可能性
- `#` （Pythonコメント）→ YAML コメントとして解釈される可能性

**特に危険な箇所:** `discovery_fallback.py` 内の `discussion_comment` フィールド:
```python
"discussion_comment": "## 🔍 要件探索\n\n以下について教えてください：\n\n1. ..."
```
この `\n` と `#` が YAML `run: |` ブロック内で問題を起こす可能性は低い（literalブロックなので）が、`setup actions` のテンプレートエンジンが処理する際に問題となる可能性がある。

#### 候補C: ファイルサイズ制限

GitHub Actions のワークフローファイルに明示的なサイズ制限はないが、極端に大きいファイルでは GitHub UI のファイルチェックが警告を出す場合がある。1492行は通常範囲内だが、内容の複雑さ（多重ネストされたheredoc内Python）が問題になる可能性。

### 4.4 調査・修正方針

```bash
# 手動で 1492行版をテスト: setup actions の出力を確認
cd /tmp/tf-test-repo
teraflow setup actions --force --dry-run 2>&1 | head -20

# YAML 検証
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/teraflow-req-agent.yml'))"

# GitHub Actions YAML lint
actionlint .github/workflows/teraflow-req-agent.yml
```

## 5. 1198行版→1492行版の移行戦略

### 5.1 選択肢

| 方式 | 説明 | リスク | 推奨 |
|------|------|--------|------|
| A: 一括移行 | `setup actions --force` のバグを修正し、1492行版を一括デプロイ | 問題3のバグ修正が必要 | ✅ **推奨（問題3修正後）** |
| B: 段階的パッチ | 1198行版に手動で差分を適用 | パッチ漏れリスク、メンテナンス二重化 | △ 応急処置としてのみ |
| C: 手動コピー | テンプレートをレンダリングして手動でteraflow-checkにコピー | 一回限り、次のsetup actionsで上書きされる | △ 検証用 |

### 5.2 推奨移行計画

```text
Step 1: 問題3の原因特定（setup actionsバグ調査）
  ├── setup actions --force の出力を手動検査
  ├── YAML lint で問題箇所を特定
  └── テンプレートエンジンのデバッグ

Step 2: テンプレートバグ修正
  ├── PYEOF/heredoc/エスケープ問題を修正
  └── setup actions --force --dry-run で検証

Step 3: teraflow-checkへデプロイ
  ├── setup actions --force で 1492行版を展開
  └── YAML lint + GitHub Actions UI で検証

Step 4: E2E再テスト
  ├── bash scripts/e2e-discovery.sh --version v0.5.18
  └── Phase C/D の pass 確認
```

### 5.3 応急処置（Step 1-2 の間）

問題3の修正に時間がかかる場合、**方式C（手動コピー）**で一時的にデプロイ:

```bash
# テンプレートからワークフローを手動レンダリング
cd /Users/taka-sho/Documents/github.com/taka-sho/teraflow
sed "s/{{.TeraflowVersion}}/v0.5.17/g" \
  internal/actions/templates/teraflow-req-agent.yml \
  > /tmp/rendered-workflow.yml

# YAML検証
python3 -c "import yaml; yaml.safe_load(open('/tmp/rendered-workflow.yml'))"

# teraflow-checkにコピー
cp /tmp/rendered-workflow.yml \
  /Users/taka-sho/Documents/github.com/taka-sho/teraflow-check/.github/workflows/teraflow-req-agent.yml

# commit + push
cd /Users/taka-sho/Documents/github.com/taka-sho/teraflow-check
git add .github/workflows/teraflow-req-agent.yml
git commit -m "chore: update teraflow-req-agent.yml to 1492-line version with parse logic"
git push origin main
```

## 6. E2Eスクリプト修正

### 6.1 V6 検証の修正（replies 対応）

```bash
# 現行: top-level のみ
COMMENTS=$(gh api graphql -f query='
  query(...) {
    repository(...) {
      discussion(number: $number) {
        comments(last: 5) {
          nodes { author { login } body createdAt }
        }
      }
    }
  }' ...)

# 修正: replies を含む
COMMENTS=$(gh api graphql -f query='
  query(...) {
    repository(...) {
      discussion(number: $number) {
        comments(last: 10) {
          nodes {
            author { login } body createdAt
            replies(last: 10) {
              nodes { author { login } body createdAt }
            }
          }
        }
      }
    }
  }' ...)
```

### 6.2 state/summary 取得の改善

```bash
# 現行: ローカルcloneを git pull で更新
cd "$WORK_DIR" && git pull origin main

# 改善: 各ラウンド後に明示的に最新を取得
cd "$WORK_DIR" && git fetch origin main && git reset --hard origin/main
```

## 7. サブタスク分解

| # | タスク | 対象リポ | 依存 | サイズ | 問題 |
|---|--------|---------|------|--------|------|
| T1 | setup actions --force のバグ調査・修正 | teraflow | なし | M | P3 |
| T2 | 1492行版テンプレートの YAML lint 検証 | teraflow | T1 | S | P3 |
| T3 | teraflow-check への 1492行版デプロイ（setup actions or 手動） | teraflow-check | T1 or 応急 | S | P1 |
| T4 | E2Eスクリプト V6 修正（replies 対応） | multi-agent-shogun | なし | S | E2E |
| T5 | E2Eスクリプト state/summary 取得改善 | multi-agent-shogun | なし | S | E2E |
| T6 | confirmed=[] 時の doc generate ガード追加 | teraflow | なし | S | P2 |
| T7 | E2E再テスト（v0.5.18 or デプロイ後） | multi-agent-shogun | T3,T4,T5 | M | 全体 |

### 7.1 推奨実装順序

```text
Batch 1（並行可能。即時着手）:
  T1（setup actionsバグ調査）
  T4 + T5（E2Eスクリプト修正。T1と並行）

Batch 2（T1完了後）:
  T2（YAML lint検証）→ T3（デプロイ）

Batch 2'（T1が長期化する場合の応急処置）:
  T3（手動コピーでデプロイ）

Batch 3（デプロイ後）:
  T6（doc generate ガード）
  T7（E2E再テスト）
```

### 7.2 最小リリース単位

**T3（手動コピーによる応急デプロイ）+ T4（V6修正）**で Phase C/D の主要問題が解消する。

- T3 により 1492行版が稼働 → パースロジック有効化 → confirmed 蓄積開始
- T4 により E2E の V6 検証が replies を検出 → round 検証が正しく動作
