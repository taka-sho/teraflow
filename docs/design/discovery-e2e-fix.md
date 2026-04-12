---
codd:
  node_id: "design:discovery-e2e-fix"
  title: "Discovery E2Eテスト発覚問題の修正設計書"
  depends_on:
    - id: "design:requirements-discovery"
      relation: extends
    - id: "design:discovery-multiround"
      relation: extends
  tags:
    - discovery
    - e2e-fix
    - multiround
  status: draft
---

# Discovery E2Eテスト発覚問題の修正設計書

## 1. 問題の全体像: カスケード障害

E2Eテスト（v0.5.15、Discussion #58）で発覚した3問題は**独立ではなく、P1を起点としたカスケード障害**である。

```text
P1: teraflow agent assign --skill discovery が失敗
  │  → AI応答がJSON不正 or コマンド自体のエラー
  │  → new_branches 空 → 個別質問ノード未生成
  │  → tree に dialogue.{ts} ノードのみ蓄積
  │
  ├──→ P2: confirmed 蓄積が動作しない
  │     → stage1_parse_user_answers は non-dialogue ノードを検索
  │     → non-dialogue ノードが0件 → parse_result 空
  │     → confirmed 空のまま
  │
  └──→ P3: CoDD document 生成エラー
        → teraflow doc generate は discovery state + confirmed を入力とする
        → confirmed 空 + tree にまともなデータなし → 生成失敗
```

**修正優先順位: P1 が最重要。P1 が解決すれば P2 は自動的に動作し、P3 も前提データが揃う。**

## 2. 問題1: 質問ツリーが正常に構築されない

### 2.1 症状

Discussion #58 の state ファイル:
```yaml
tree:
  - id: dialogue.1776007116484
    question: "要件探索の実行に失敗しました。"
    category: dialogue
    meta:
      assistant_comment: "要件探索の実行に失敗しました。"
```

全ノードが `category: dialogue` で `question: "要件探索の実行に失敗しました。"` という同一パターン。

### 2.2 根本原因分析

**原因チェーン:**

```text
[1] teraflow agent assign --skill discovery が実行される
     ↓
[2] LLM（assignments設定のモデル）が応答を生成
     ↓
[3] 応答がJSON不正 → parse_payload がフォールバック
     → data = {"discussion_comment": raw_text}
     → new_branches, resolved_branches, auto_resolved 全て空
     ↓
[4] 空のnew_branchesで個別ノードが生成されない
     ↓
[5] フォールバックコメント判定(count_numbered_questions < 3)
     → build_fallback_comment が呼ばれる
     ↓
[6] しかしワークフローステップのエラーハンドリング（544-548行目）:
     RC != 0 の場合 → '{"output":"要件探索の実行に失敗しました。"}' を書き出し
     → comment = "要件探索の実行に失敗しました。"
     → is_footer_only = False（エラー文言は質問リストではないがfooterでもない）
     → build_fallback_comment は呼ばれない
     ↓
[7] comment = "要件探索の実行に失敗しました。" がそのまま dialogue ノードに保存
```

**真の根本原因は2つの層にある:**

| 層 | 原因 | 詳細 |
|----|------|------|
| **Layer A: agent assign 失敗** | `teraflow agent assign --skill discovery` がエラー終了（RC≠0） | モデルがJSON出力に失敗、APIキー問題、またはagent assignコマンド自体のバグ |
| **Layer B: エラー時のフォールバック不足** | RC≠0 時に `"要件探索の実行に失敗しました。"` を返すだけ | フォールバック質問リストが生成されないため、ユーザーに質問が届かない |

### 2.3 Layer A の詳細分析: agent assign が失敗する原因

#### 候補1: LLMモデルのJSON遵守率
- `skills/discovery.yml` は strict JSON 出力を要求（コードフェンス禁止、前置き禁止）
- gpt-4o-mini は指示遵守率が低く、マークダウンコードフェンスや前置きテキストを付けがち
- `parse_payload` (571-605行目) はコードフェンス内JSON抽出や `{` ~ `}` 範囲抽出を試みるが、LLM出力が完全に非JSON（純テキスト応答）だった場合は失敗する

#### 候補2: agent assign コマンドのエラー
- `teraflow agent assign` がスキル定義読み込みやAPI呼び出しで失敗
- E2Eテストでは Phase B pass（初回応答あり）なので、少なくとも初回は動作している
- ただし初回の `discussion_comment` も「要件探索の実行に失敗しました。」だった可能性

#### 候補3: 入力サイズ超過
- ラウンドが進むと discovery_input.txt が肥大化
- 構造化サマリ + Discussion本文 + 最新コメントがモデルのコンテキスト制限を超える

**最も可能性が高い原因: 候補1（LLMのJSON遵守率）+ 候補2のエラーハンドリング不足の複合。**

### 2.4 修正方針

#### Fix 1-A: agent assign エラー時のフォールバック質問生成

```python
# 現行（544-548行目）
if [ "${RC:-0}" != "0" ]; then
    echo '{"output":"要件探索の実行に失敗しました。"}' > /tmp/discovery_result.json
fi

# 修正後
if [ "${RC:-0}" != "0" ]; then
    # エラー時でもフォールバック質問を含む有効なJSONを返す
    python3 -c "
import json
fallback = {
    'output': json.dumps({
        'action': 'ask',
        'resolved_branches': [],
        'new_branches': [
            {'id': 'scope.target_users', 'question': '対象ユーザーはどれですか？', 'category': 'scope', 'recommendation': '社内利用者'},
            {'id': 'scope.platform', 'question': '対応プラットフォームはどれですか？', 'category': 'scope', 'recommendation': 'Webのみ'},
            {'id': 'scope.priority', 'question': '最優先で実装する範囲はどれですか？', 'category': 'priority', 'recommendation': 'MVP最小構成'},
        ],
        'auto_resolved': [],
        'next_question': {'id': 'scope.target_users', 'question': '対象ユーザーはどれですか？', 'recommendation': '社内利用者'},
        'draft_update': '',
        'discussion_comment': '## 🔍 要件探索\n\n以下について教えてください：\n\n1. 対象ユーザーはどれですか？\n   a) 社内利用者  b) 一般ユーザー  c) 管理者  d) その他\n\n2. 対応プラットフォームはどれですか？\n   a) Webのみ  b) Web+API  c) モバイル含む  d) 未定\n\n3. 最優先で実装する範囲はどれですか？\n   a) MVP最小構成  b) 標準機能一式  c) 拡張含む  d) 未定\n\n> 記号で回答できます: 「1a 2b 3c」'
    }, ensure_ascii=False)
}
json.dump(fallback, open('/tmp/discovery_result.json', 'w'), ensure_ascii=False)
" 2>/dev/null || echo '{"output":"要件探索の実行に失敗しました。"}' > /tmp/discovery_result.json
fi
```

**効果:** agent assign が失敗しても、ユーザーには有効な質問リストが表示され、tree に個別質問ノード（scope.target_users等）が生成される。

#### Fix 1-B: parse_payload の堅牢化

```python
# 現行のフォールバック（605行目）
return {"discussion_comment": text}

# 修正: JSONパース全失敗時もnew_branchesを生成
def parse_payload(raw):
    # ... 既存のJSONパース試行 ...

    # 全パース失敗 → テキストをdiscussion_commentとして扱い、
    # かつフォールバックのnew_branchesを含める
    return {
        "discussion_comment": text,
        "new_branches": [],  # 明示的に空配列（後でbuild_fallback_commentが処理）
    }
```

#### Fix 1-C: build_fallback_comment の判定条件修正

```python
# 現行（840-844行目）
requires_question_list = skill_mode != "confirm"
if requires_question_list:
    question_count = count_numbered_questions(comment)
    if question_count < 3 or is_footer_only(comment):
        comment = build_fallback_comment()

# 修正: エラーメッセージ検出を追加
requires_question_list = skill_mode != "confirm"
if requires_question_list:
    question_count = count_numbered_questions(comment)
    is_error_message = any(kw in comment for kw in ["失敗しました", "エラー", "error", "failed"])
    if question_count < 3 or is_footer_only(comment) or is_error_message:
        comment = build_fallback_comment()
```

**効果:** エラーメッセージがcommentに入った場合でも、フォールバック質問リストに差し替わる。

#### Fix 1-D: Discussion本文からの自動new_branches生成

agent assign失敗時、Discussion本文のキーワードから初期質問を動的生成する:

```python
def generate_fallback_branches_from_discussion(disc_body, disc_title):
    """Discussion本文から推測して初期new_branchesを生成する。"""
    branches = [
        {"id": "scope.target_users", "question": "対象ユーザーはどれですか？",
         "category": "scope", "recommendation": "社内利用者"},
        {"id": "scope.platform", "question": "対応プラットフォームはどれですか？",
         "category": "scope", "recommendation": "Webのみ"},
        {"id": "prio.phase1", "question": "Phase 1の範囲はどれですか？",
         "category": "priority", "recommendation": "MVP最小構成"},
    ]
    # Discussion本文にキーワードがあれば追加質問を生成
    body_lower = (disc_body or "").lower()
    if any(kw in body_lower for kw in ["認証", "auth", "ログイン", "login"]):
        branches.append({"id": "func.auth_method", "question": "認証方式はどれですか？",
                         "category": "functional", "recommendation": "OAuth 2.0 + JWT"})
    if any(kw in body_lower for kw in ["api", "エンドポイント", "rest"]):
        branches.append({"id": "func.api_style", "question": "API方式はどれですか？",
                         "category": "functional", "recommendation": "REST API"})
    return branches
```

## 3. 問題2: cmd_216の回答蓄積がワークフロー環境で動作していない

### 3.1 症状

E2Eレポート:
```yaml
V7_confirmed_count: "fail (0 < 1)"
V9_answered_nodes: "fail (0 < 1)"
```
round_count=2まで進んでも confirmed=[] のまま。

### 3.2 根本原因

**P1 の直接的帰結。** パースロジック自体は正しく実装されている（740-876行目で確認済み）が、**パース対象の非dialogueノードが存在しない**ため動作しない。

```text
stage1_parse_user_answers("1a 2b 3a", tree)
  ↓
find_recent_assistant_comment(tree)
  → "要件探索の実行に失敗しました。"  ← 質問リストではない
  ↓
extract_questions_from_comment("要件探索の実行に失敗しました。")
  → {} (質問パターンに一致する行なし)
  ↓
parse_user_answer("1a 2b 3a", "要件探索の実行に失敗しました。")
  → {1: "a", 2: "b", 3: "a"} ← パース自体は成功
  ↓
find_question_node(1, {}, tree)
  → non_dialogue = [] ← 非dialogueノードが0件
  → ordered_pending = [] ← pending非dialogueノードが0件
  → return None ← マッチなし
  ↓
parse_result = [] ← 空
confirmed = [] ← 空のまま
```

### 3.3 修正方針

#### Fix 2-A: フォールバック質問ノードからのパース（P1修正の補完）

P1の修正（Fix 1-A〜1-D）により、agent assign失敗時でも個別質問ノード（scope.target_users等）がtreeに追加される。これにより`find_question_node`がノードを見つけられるようになり、パースロジックが正常に動作する。

**追加修正は不要。P1修正で自動解消。**

#### Fix 2-B: 防御的パース — dialogueノードのassistant_commentからのフォールバック

P1が完全に修正できない場合の保険:

```python
def stage1_parse_user_answers(user_text, nodes):
    prev_comment = find_recent_assistant_comment(nodes)
    questions = extract_questions_from_comment(prev_comment)
    parsed_answers = parse_user_answer(user_text, prev_comment)
    parse_result = []

    for q_num, answer_code in sorted(parsed_answers.items()):
        q_info = questions.get(q_num) or {"text": "", "choices": {}, "category": "unknown"}
        node = find_question_node(q_num, q_info, nodes)

        # NEW: ノードが見つからない場合、質問テキストから仮ノードを生成
        if node is None and q_info.get("text"):
            node_id = f"fallback.q{q_num}"
            node = {
                "id": node_id,
                "question": q_info["text"],
                "category": q_info.get("category") or "unknown",
                "status": "pending",
                "recommendation": "",
                "answer": "",
                "children": [],
            }
            nodes.append(node)  # tree に追加

        if node is None:
            continue
        # ... 以降は既存ロジック
```

**効果:** assistant_commentに質問リストが含まれていれば（build_fallback_commentで生成されたものでも）、対応するノードを動的生成してパースを成功させる。

## 4. 問題3: 「要求確定」時のCoDD document生成エラー

### 4.1 症状

E2Eレポート:
```yaml
D_confirm:
  result: fail
  verifications:
    V12_confirm_response: "pending"
    ...
  errors:
    step: "workflow_wait"
    message: "workflow failed after confirm"
```

### 4.2 根本原因

`teraflow doc generate --discussion 58` が失敗する。

**原因チェーン:**

```text
[1] teraflow doc generate が呼ばれる（1258行目）
     ↓
[2] discovery state ファイルを読む
     → tree に dialogue ノードしかない
     → confirmed 空、有意な要件データなし
     ↓
[3] doc generate が有意なコンテンツを生成できずエラー
     OR ドラフトファイルが最小限（"# discovery draft #58" のみ）でCoDD frontmatter不足
     ↓
[4] RC≠0 → "FAILED_STEP=Generate CoDD document"
     → Discussion に「❌ エラーが発生しました」を投稿
```

### 4.3 修正方針

#### Fix 3-A: doc generate のエラーハンドリング改善

```bash
# 現行（1264-1268行目）
if [ "${RC:-0}" != "0" ]; then
    echo "FAILED_STEP=Generate CoDD document" >> $GITHUB_ENV
    echo "ERROR_TYPE=teraflow" >> $GITHUB_ENV
    echo "ERROR_FILE=/tmp/teraflow_docgen.err" >> $GITHUB_ENV
    exit $RC  # ← ワークフロー全体が失敗
fi

# 修正: フォールバックでドラフトからPRを生成
if [ "${RC:-0}" != "0" ]; then
    # doc generate 失敗時、discovery draftが存在すればそれをベースにPRを作成
    DRAFT_FILE=".teraflow/discovery/discussion-${DISC_NUM}-draft.md"
    if [ -f "$DRAFT_FILE" ]; then
        BRANCH="docs/req-discussion-${DISC_NUM}"
        git checkout -b "$BRANCH" 2>/dev/null || git checkout "$BRANCH"
        mkdir -p docs/requirements
        cp "$DRAFT_FILE" "docs/requirements/discussion-${DISC_NUM}.md"
        git add "docs/requirements/discussion-${DISC_NUM}.md"
        git commit -m "docs(requirements): add requirement draft from discussion #${DISC_NUM}"
        git push origin "$BRANCH"
        PR_URL=$(gh pr create \
            --title "docs: requirement draft from discussion #${DISC_NUM}" \
            --body "Discovery ドラフトからの自動生成（doc generate フォールバック）" \
            --base main \
            --head "$BRANCH" \
            2>/dev/null | grep -o 'https://[^ ]*')
        echo "pr_url=$PR_URL" >> $GITHUB_OUTPUT
    else
        echo "FAILED_STEP=Generate CoDD document" >> $GITHUB_ENV
        echo "ERROR_TYPE=teraflow" >> $GITHUB_ENV
        echo "ERROR_FILE=/tmp/teraflow_docgen.err" >> $GITHUB_ENV
        # exit $RC を削除 — doc generate失敗でもワークフロー全体は失敗させない
    fi
fi
```

#### Fix 3-B: discovery-confirm モード時のドラフト品質保証

```python
# 「要求確定」時、confirmed が空の場合の警告コメント生成
if skill_mode == "confirm" and not confirmed:
    comment = ("## ⚠️ 確定可能な要件がありません\n\n"
               "まだ質問への回答が蓄積されていません。\n"
               "先に質問に回答してから「要求確定」を実行してください。\n\n"
               "> 記号で回答できます: 「1a 2b 3c」")
    # doc generate をスキップ
    skip_docgen = True
```

**効果:** confirmed が空の状態で doc generate を呼ばず、ユーザーに適切なフィードバックを返す。

## 5. 修正の優先順位と依存関係

```text
╔═══════════════════════════════════════════════════╗
║  FIX PRIORITY MAP                                 ║
╠═══════════════════════════════════════════════════╣
║                                                   ║
║  [P0] Fix 1-A: agent assign エラー時フォールバック ║
║    │   → tree に個別ノード生成を保証              ║
║    │                                               ║
║    ├── [P1] Fix 1-C: build_fallback_comment       ║
║    │   判定条件修正（エラーメッセージ検出）        ║
║    │                                               ║
║    ├── [P1] Fix 2-B: 防御的パース                 ║
║    │   （ノード未発見時の仮ノード生成）            ║
║    │                                               ║
║    ├── [P2] Fix 3-A: doc generate エラー          ║
║    │   ハンドリング（フォールバックPR）            ║
║    │                                               ║
║    └── [P2] Fix 3-B: confirmed空時の               ║
║        doc generate スキップ                       ║
║                                                   ║
║  [P1] Fix 1-B: parse_payload 堅牢化               ║
║  [P1] Fix 1-D: Discussion本文からの動的            ║
║       フォールバックbranch生成                     ║
║                                                   ║
╚═══════════════════════════════════════════════════╝
```

## 6. サブタスク分解

| # | タスク | 修正対象 | 依存 | サイズ | P1/P2/P3 |
|---|--------|---------|------|--------|----------|
| T1 | agent assign エラー時フォールバック質問JSON生成 | teraflow-req-agent.yml 544-548行 | なし | S | P1 |
| T2 | build_fallback_comment のエラーメッセージ検出追加 | teraflow-req-agent.yml 840-844行 | T1 | S | P1 |
| T3 | 防御的パース: ノード未発見時の仮ノード生成 | teraflow-req-agent.yml stage1_parse_user_answers | T1 | S | P2 |
| T4 | confirmed 空時の doc generate スキップ + 警告コメント | teraflow-req-agent.yml discovery-confirm分岐 | T1 | S | P3 |
| T5 | doc generate エラー時のフォールバックPR生成 | teraflow-req-agent.yml 1264-1268行 | T4 | M | P3 |
| T6 | Discussion本文からの動的フォールバックbranch生成 | teraflow-req-agent.yml 新関数 | T1 | S | P1強化 |
| T7 | parse_payload フォールバック改善 | teraflow-req-agent.yml 605行 | なし | S | P1強化 |
| T8 | E2Eテスト再実行で検証 | scripts/e2e-discovery.sh | T1-T5 | M | 全体 |

S = 1セッション, M = 2セッション

### 6.1 推奨実装順序

```text
Batch 1（最重要。これだけでP1の主要問題を解消）:
  T1 → T2（agent assign フォールバック + build_fallback_comment 修正）

Batch 2（P2/P3対応）:
  T3 + T4（並行実行可能。防御的パース + confirm時スキップ）

Batch 3（堅牢化）:
  T5 + T6 + T7（並行実行可能。doc generate改善 + 動的branch + parse_payload）

Batch 4（検証）:
  T8（E2E再実行）
```

### 6.2 最小リリース単位

**T1 + T2 のみで最大の効果が得られる。** agent assign 失敗時にフォールバック質問ノードが生成されれば:
- P1: ユーザーに質問が届く（✅解消）
- P2: 個別ノードが存在するのでパースが動作する（✅解消）
- P3: confirmed が蓄積されるので doc generate の入力が揃う（✅大幅改善）

### 6.3 根本対策（中長期）

| 対策 | 説明 | 優先度 |
|------|------|--------|
| agent assign の失敗原因調査 | workflow run ログから実際のエラー内容を特定 | HIGH |
| LLMモデルの切り替え検討 | gpt-4o-mini → claude-haiku-4-5 等、JSON遵守率が高いモデルへ | MEDIUM |
| agent assign のリトライ機構 | 1回失敗した場合、別モデルでリトライ | LOW |
| discovery mode のローカルテスト | `teraflow agent assign --skill discovery --dry-run` でCI外テスト | HIGH |
