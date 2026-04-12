---
codd:
  node_id: "design:discovery-multiround"
  title: "Discovery 複数ラウンド会話の回答蓄積・参照 設計書"
  depends_on:
    - id: "design:requirements-discovery"
      relation: extends
  tags:
    - discovery
    - multiround
    - conversation
    - state-management
  status: draft
---

# Discovery 複数ラウンド会話の回答蓄積・参照 設計書

## 1. 根本原因分析

### 1.1 問題の概要

Discussion #51（teraflow-check）で確認された3つの問題:

| # | 症状 | 影響 |
|---|------|------|
| P1 | ユーザー回答（「1a 2b」等）が summary.yaml の confirmed に反映されない | 確定事項が蓄積されず、次ターンでAIが参照できない |
| P2 | tree が個別質問ブランチではなく `dialogue.{timestamp}` 汎用ノードで管理される | 質問単位の状態追跡ができず、進捗計算が無意味になる |
| P3 | 次ターンのAI入力で confirmed が空のまま渡される | AIが前回確定事項を知らず、同じ質問を繰り返す |

### 1.2 根本原因 1: ユーザー回答パースの欠如

**現行コード** (`teraflow-req-agent.yml` 722-743行目):

```python
# 毎ターン dialogue.{timestamp} ノードを無条件追加
turn_id = f"dialogue.{int(datetime.now(timezone.utc).timestamp() * 1000)}"
user_input = disc_body if skill_mode == "init" else latest_comment
tree.append({
    "id": turn_id,
    "question": pick_digest(comment, "要件探索の問いかけ"),
    "category": "dialogue",
    "status": "answered" if user_input else "pending",
    "answer": user_input,  # ← ユーザーの生コメントをそのまま格納
    ...
})
```

**問題点:**
- ユーザーが「1a 2b 3d」と回答しても、それが**どの質問のどの選択肢**に対する回答かを解析しない
- 生コメントを丸ごと1つの `dialogue` ノードに格納するだけ
- AIが返す `resolved_branches` は、ユーザー回答のパースに依存するが、AI 側もパースロジックを持たない

**結果:** `resolved_branches` が空 → tree内の個別質問ノードが `answered` にならない → confirmed リストに追加されない

### 1.3 根本原因 2: tree構造の問題（dialogue.{timestamp}ノード方式）

**現行の tree 構造:**

```yaml
tree:
  # AIが new_branches で追加した個別質問ノード（正しい構造）
  - id: "scope.target_users"
    question: "対象ユーザーはどれですか？"
    status: "pending"  # ← ずっと pending のまま

  # 毎ターン追加される dialogue ノード（問題の構造）
  - id: "dialogue.1712956800000"
    category: "dialogue"
    status: "answered"
    answer: "1a 2b"  # ← 生コメント。どの質問への回答か不明
    meta:
      assistant_comment: "## 🔍 要件探索\n1. 対象ユーザーは..."
```

**問題点:**
- 個別質問ノード（`scope.target_users` 等）と `dialogue` ノードが**切り離されている**
- `dialogue` ノードは `category: "dialogue"` のため `recompute_summary` で除外される（846-849行目）
- 個別質問ノードは `pending` のまま → `confirmed` に入らない → AIが再質問

**本来あるべき流れ:**
```
AIが質問生成 → new_branches で個別ノード追加
  ↓
ユーザーが「1a」と回答
  ↓
「1a」をパースし、該当ノードを answered に更新  ← ★ ここが欠落
  ↓
confirmed に反映 → 次ターンでAIが参照
```

### 1.4 根本原因 3: confirmed自動更新ロジックの不完全性

**現行の summary 更新ロジック** (`teraflow-req-agent.yml` 900-920行目):

```python
for node in walk(tree):
    if (node.get("category") or "").strip().lower() == "dialogue":
        continue  # ← dialogue ノードは除外（正しい）
    status = (node.get("status") or "").strip().lower()
    if status == "answered" and answer:
        confirmed.append(f"{question}: {answer}")  # ← 個別ノードが answered なら追加
```

**このロジック自体は正しい。** 問題は、**個別ノードが answered にならない**こと。

原因チェーン:
```
ユーザー回答パース欠如 (原因1)
  → resolved_branches が空
    → mark_resolved が呼ばれない
      → 個別ノードが pending のまま
        → confirmed が空
          → AI入力にconfirmedなし (問題P3)
            → AIが再質問 (問題P1)
```

### 1.5 原因の依存関係図

```text
[ユーザー回答パース欠如] ──────────────────────┐
        │                                       │
        ▼                                       ▼
[resolved_branches 空] ──→ [個別ノード pending のまま]
        │                           │
        ▼                           ▼
[confirmed リスト空] ──────→ [AIが再質問]
        │
        ▼
[dialogue.{ts} に生コメント蓄積] ──→ [tree肥大化 / 構造崩壊]
```

**修正の優先順位:** パースロジック追加 (原因1) が最優先。これが解決すれば原因2,3は連鎖的に解消する。

## 2. ユーザー回答パースロジック設計

### 2.1 対応すべき回答パターン

| パターン | 例 | 解析結果 |
|---------|-----|---------|
| **番号+選択肢** | `1a 2b 3d` | Q1→選択肢a, Q2→選択肢b, Q3→選択肢d |
| **番号+ドット+選択肢** | `1.a 2.b` | Q1→選択肢a, Q2→選択肢b |
| **カンマ区切り** | `1a, 2b, 3c` | Q1→選択肢a, Q2→選択肢b, Q3→選択肢c |
| **スラッシュ区切り** | `1. a / 2. a,b` | Q1→選択肢a, Q2→選択肢a,b |
| **OK** | `OK` / `ok` / `はい` | 全 pending 質問を推奨回答で承認 |
| **スキップ** | `スキップ` / `skip` | 全 pending 質問をスキップ |
| **自由記述** | `JWTの有効期限は24時間にしたい` | パース不可 → AI解析に委譲 |
| **混合** | `1a 2は24時間で` | Q1→選択肢a, Q2→自由記述「24時間で」 |

### 2.2 パースアルゴリズム

```python
import re

def parse_user_answer(comment: str, questions: list[dict]) -> list[dict]:
    """
    ユーザーコメントを解析し、各質問への回答を抽出する。

    Args:
        comment: ユーザーのコメント本文
        questions: 直前のAIコメントから抽出した質問リスト
                   [{"number": 1, "branch_id": "scope.target_users",
                     "choices": {"a": "社内利用者", "b": "一般ユーザー", ...}}]

    Returns:
        [{"branch_id": "scope.target_users", "answer": "社内利用者",
          "resolved_by": "user", "choice_key": "a"}]
    """
    text = comment.strip()

    # 特殊コマンド判定
    if re.match(r'^(OK|ok|はい|承認)$', text):
        return [{"branch_id": q["branch_id"],
                 "answer": q.get("recommendation", ""),
                 "resolved_by": "ai-default"}
                for q in questions if q.get("recommendation")]

    if re.match(r'^(スキップ|skip|後回し)$', text, re.IGNORECASE):
        return [{"branch_id": q["branch_id"],
                 "answer": "", "resolved_by": "skipped",
                 "skip_reason": "ユーザーがスキップを選択"}
                for q in questions]

    # 番号+選択肢パターンの抽出
    # "1a 2b 3d" / "1.a 2.b" / "1a, 2b" / "1. a / 2. b"
    pattern = r'(\d+)[\.\s]*([a-z])(?:\s*[,/]\s*|\s+|$)'
    matches = re.findall(pattern, text.lower())

    results = []
    matched_numbers = set()

    for num_str, choice_key in matches:
        num = int(num_str)
        matched_numbers.add(num)
        q = next((q for q in questions if q["number"] == num), None)
        if q is None:
            continue
        choice_text = q.get("choices", {}).get(choice_key, "")
        results.append({
            "branch_id": q["branch_id"],
            "answer": choice_text if choice_text else f"選択肢{choice_key}",
            "resolved_by": "user",
            "choice_key": choice_key,
        })

    # 番号+自由記述パターン: "2は24時間で"
    free_pattern = r'(\d+)[\.\s]*(?:は|:)\s*(.+?)(?=\d+[\.\s]*[a-zは:]|\s*$)'
    free_matches = re.findall(free_pattern, text)
    for num_str, free_text in free_matches:
        num = int(num_str)
        if num in matched_numbers:
            continue
        matched_numbers.add(num)
        q = next((q for q in questions if q["number"] == num), None)
        if q is None:
            continue
        results.append({
            "branch_id": q["branch_id"],
            "answer": free_text.strip(),
            "resolved_by": "user",
        })

    return results
```

### 2.3 直前AIコメントからの質問抽出

パースには「直前のAIコメントでどんな質問が提示されたか」が必要。

**抽出ロジック:**

```python
def extract_questions_from_comment(assistant_comment: str, tree: list) -> list[dict]:
    """
    AIが投稿したコメントから質問番号・選択肢・対応ブランチIDを抽出する。

    例: "1. 対象ユーザーはどれですか？\n   a) 社内利用者  b) 一般ユーザー"
    → [{"number": 1, "branch_id": "scope.target_users",
         "question": "対象ユーザーはどれですか？",
         "choices": {"a": "社内利用者", "b": "一般ユーザー"},
         "recommendation": "社内利用者"}]
    """
    questions = []
    lines = assistant_comment.split("\n")
    current_q = None

    for line in lines:
        # 番号付き質問行: "1. 対象ユーザーは..."
        q_match = re.match(r'^(\d+)[\.\)]\s+(.+)', line.strip())
        if q_match:
            if current_q:
                questions.append(current_q)
            num = int(q_match.group(1))
            question_text = q_match.group(2).strip()
            # tree内のpendingノードとマッチング
            branch_id = match_question_to_branch(question_text, tree)
            current_q = {
                "number": num,
                "question": question_text,
                "branch_id": branch_id,
                "choices": {},
                "recommendation": "",
            }
            continue

        # 選択肢行: "   a) 社内利用者  b) 一般ユーザー"
        if current_q:
            choice_matches = re.findall(r'([a-z])\)\s*([^a-z\)]+?)(?=\s+[a-z]\)|\s*$)', line.strip())
            for key, text in choice_matches:
                current_q["choices"][key] = text.strip()
            # 最初の選択肢を推奨回答とする
            if current_q["choices"] and not current_q["recommendation"]:
                first_key = sorted(current_q["choices"].keys())[0]
                current_q["recommendation"] = current_q["choices"][first_key]

    if current_q:
        questions.append(current_q)

    return questions


def match_question_to_branch(question_text: str, tree: list) -> str:
    """
    質問テキストと tree 内の pending ノードをマッチングし、branch_id を返す。
    完全一致 → 部分一致 → 見つからなければ空文字列。
    """
    for node in walk(tree):
        if node.get("category") == "dialogue":
            continue
        if (node.get("status") or "pending") != "pending":
            continue
        node_q = (node.get("question") or "").strip()
        if node_q == question_text:
            return node.get("id", "")
        # 部分一致（質問文の先頭30文字が一致）
        if len(node_q) > 10 and node_q[:30] == question_text[:30]:
            return node.get("id", "")
    return ""
```

### 2.4 パース結果の適用フロー

```text
ユーザーコメント受信
    │
    ▼
[1] 前ターンの dialogue ノードから assistant_comment を取得
    │
    ▼
[2] assistant_comment から質問リスト抽出
    （extract_questions_from_comment）
    │
    ▼
[3] ユーザーコメントをパース
    （parse_user_answer）
    │
    ▼
[4] パース成功した質問 → tree の該当ノードを answered に更新
    パース失敗した質問 → AI解析に委譲（resolved_branches で処理）
    │
    ▼
[5] confirmed リスト更新 → summary.yaml に書き出し
```

## 3. tree構造改善設計

### 3.1 方針: dialogue ノードの段階的廃止

**互換性を維持しつつ、dialogue ノードの役割を縮小する。**

| 現行 | 改善後 |
|------|--------|
| 毎ターン `dialogue.{ts}` ノード追加 | 会話ログとしてのみ残す（meta 情報格納用） |
| 個別質問ノードが pending のまま放置 | パース結果で answered に更新 |
| dialogue ノードの answer に生コメント格納 | 個別ノードの answer に構造化された回答を格納 |

### 3.2 改善後の tree 構造

```yaml
tree:
  # 個別質問ノード（AIが new_branches で追加。パースで answered に更新される）
  - id: "scope.target_users"
    question: "対象ユーザーはどれですか？"
    category: "scope"
    status: "answered"           # ← パースにより updated
    recommendation: "社内利用者"
    answer: "社内利用者"          # ← パース結果
    resolved_by: "user"
    resolved_at: "2026-04-12T23:00:00Z"

  - id: "scope.platform"
    question: "対象プラットフォームは？"
    category: "scope"
    status: "answered"
    answer: "Web + CLI"
    resolved_by: "user"

  - id: "func.error_handling"
    question: "エラー時の振る舞いは？"
    category: "functional"
    status: "pending"            # ← 未回答

  # 会話ログノード（dialogue。meta情報格納用。confirmed/summaryからは除外）
  - id: "dialogue.1712956800000"
    category: "dialogue"
    status: "answered"
    answer: "1a 2b"              # 生コメント保持（デバッグ用）
    meta:
      assistant_comment: "..."
      latest_user_comment: "1a 2b"
      parsed_answers:            # NEW: パース結果の記録
        - branch_id: "scope.target_users"
          choice: "a"
          answer: "社内利用者"
        - branch_id: "scope.platform"
          choice: "b"
          answer: "Web + CLI"
      unparsed_portion: ""       # パースできなかった部分
```

### 3.3 new_branches → 個別ブランチ登録 → resolved_branches 処理の流れ

```text
Round 1 (init):
  AI generates new_branches → [scope.target_users, scope.platform, func.error_handling]
  tree に 3 つの pending ノード追加
  AI comment: "1. 対象ユーザーは？ a) 社内利用者 b) 一般ユーザー ..."
  dialogue ノード追加（meta に assistant_comment 記録）

Round 2 (ユーザー回答):
  ユーザー: "1a 2b"
  [パースフェーズ]
    前ターンの dialogue.meta.assistant_comment から質問抽出
    "1a" → scope.target_users を answered（answer: "社内利用者"）
    "2b" → scope.platform を answered（answer: "Web + CLI"）
  [AI解析フェーズ]
    AI に渡す入力:
      - confirmed: ["対象ユーザー: 社内利用者", "プラットフォーム: Web + CLI"]
      - unresolved: ["エラー時の振る舞いは？"]
    AI が次の質問を生成（func.error_handling に対する深掘り等）
  [dialogue ノード追加]
    parsed_answers を meta に記録
  [summary.yaml 更新]
    confirmed に 2 件追加
```

### 3.4 dialogue ノードの扱いルール

| ルール | 理由 |
|--------|------|
| `recompute_summary` から除外（既存通り） | 会話ログは進捗計算に含めない |
| `confirmed` 生成時に除外（既存通り） | 個別ノードから生成する |
| `meta.parsed_answers` を新設 | パース結果のトレーサビリティ確保 |
| tree 肥大化防止: 20ラウンド超で古い dialogue ノードを圧縮 | コンテキストウィンドウ節約 |

## 4. confirmed 自動更新ロジック設計

### 4.1 二段階更新方式

```text
Stage 1: ローカルパース（確実・即座）
  ユーザーコメント → parse_user_answer → 該当ノード answered 更新
  → confirmed リスト即時更新

Stage 2: AI解析（フォールバック）
  ローカルパースで解決できなかった質問
  → AI に resolved_branches として返してもらう
  → mark_resolved で該当ノード更新
  → confirmed リスト追加更新
```

**Stage 1 が優先。** パースで解決できればAI解析は不要（トークン節約）。

### 4.2 ユーザー回答→confirmed 変換ロジック

```python
def update_confirmed_from_parse(tree, parse_results, existing_confirmed):
    """
    パース結果から confirmed リストを更新する。

    変換ルール:
    - 選択肢回答: "{質問}: {選択肢テキスト}"
    - 自由記述: "{質問}: {ユーザー記述}"
    - AI推奨承認: "{質問}: {推奨回答}（AI推奨承認）"
    - スキップ: confirmed に入れない（unresolved に記録）
    """
    new_confirmed = list(existing_confirmed)

    for result in parse_results:
        branch_id = result["branch_id"]
        answer = result["answer"]
        resolved_by = result.get("resolved_by", "user")

        # tree からノードを取得
        node = find_node(tree, branch_id)
        if node is None:
            continue

        question = (node.get("question") or "").strip()
        if not question:
            question = branch_id

        if resolved_by == "skipped":
            continue  # スキップはconfirmedに入れない

        # 人間可読な確定文を生成
        if resolved_by == "ai-default":
            entry = f"{question}: {answer}（AI推奨承認）"
        else:
            entry = f"{question}: {answer}"

        # 重複防止
        if entry not in new_confirmed:
            new_confirmed.append(entry)

    return new_confirmed
```

### 4.3 AI解析との併用（Stage 2）

```python
def merge_ai_resolved(tree, ai_resolved_branches, confirmed):
    """
    AI が返した resolved_branches を tree に適用し、confirmed を更新する。
    ローカルパースで既に answered になったノードはスキップ。
    """
    for row in ai_resolved_branches or []:
        node_id = (row.get("id") or "").strip()
        if not node_id:
            continue
        node = find_node(tree, node_id)
        if node is None:
            continue
        # 既に answered ならスキップ（ローカルパースが優先）
        if (node.get("status") or "").lower() == "answered":
            continue
        # AI解析結果で更新
        node["status"] = "answered"
        node["answer"] = (row.get("answer") or "").strip()
        node["resolved_by"] = (row.get("resolved_by") or "user").strip()
        node["resolved_at"] = datetime.now(timezone.utc).isoformat()

        question = (node.get("question") or "").strip()
        answer = node["answer"]
        if question and answer:
            entry = f"{question}: {answer}"
            if entry not in confirmed:
                confirmed.append(entry)

    return confirmed
```

### 4.4 処理順序の統合

```text
1. ユーザーコメント受信
2. [Stage 1] ローカルパース
   a. 前ターンの assistant_comment から質問リスト抽出
   b. parse_user_answer でパース
   c. パース成功分 → tree の個別ノードを answered に更新
   d. confirmed リスト更新
3. AI に入力を渡す（confirmed 含む）
4. AI が応答を返す
5. [Stage 2] AI の resolved_branches を適用
   a. ローカルパースで未解決のノードのみ更新
   b. confirmed リスト追加更新
6. new_branches を tree に追加
7. dialogue ノード追加（meta に parsed_answers 記録）
8. summary.yaml 書き出し
9. state.yaml 書き出し
```

## 5. AI入力改善設計

### 5.1 現行のAI入力構造の問題

**現行:**
```text
## 構造化サマリ
confirmed: []          ← 常に空
unresolved: []         ← 個別ノードがないため空
context: [...]
next_focus: [...]
round_count: 2
```

**改善後:**
```text
## 構造化サマリ
confirmed:
  - "対象ユーザー: 社内利用者"
  - "プラットフォーム: Web + CLI"
  - "レスポンスタイム: p95 < 500ms（AI推奨承認）"
unresolved:
  - "エラー時の振る舞いは？"
  - "ロールバック計画は？"
context:
  - "Discussion: ユーザー認証機能"
  - "初回投稿要約: OAuth 2.0 ベースの認証機能を..."
next_focus:
  - "エラー時の振る舞いは？"
round_count: 3
```

### 5.2 AI再質問防止の制約設計

discovery.yml のシステムプロンプトに以下を追加:

```yaml
prompts:
  system: |
    # 既確認事項の再質問禁止ルール（CRITICAL）

    ## 構造化サマリの confirmed リスト
    confirmed に含まれる事項は**既にユーザーが確定した要件**である。
    これらの事項について再度質問してはならない。

    ## 制約
    1. confirmed に含まれる質問と同一または類似の質問を new_branches に含めないこと
    2. confirmed の内容を前提として、未解決事項のみに焦点を当てること
    3. confirmed の内容と矛盾する回答をユーザーが行った場合のみ、確認質問を許可する
       （その場合 discussion_comment に矛盾の指摘を含めること）
    4. unresolved に含まれる質問を優先的に取り上げること
    5. next_focus がある場合、それを最優先で質問すること

    ## confirmed の活用方法
    - 新しい質問の recommendation を生成する際、confirmed の内容を参照して整合性を確保すること
    - draft_update を生成する際、confirmed の内容を該当セクションに反映すること
```

### 5.3 AI入力のトークン効率化

confirmed リストが肥大化した場合の対策:

| confirmed 件数 | 対策 |
|---------------|------|
| 0-20件 | 全件を AI 入力に含める |
| 21-40件 | カテゴリ別に集約（例: "スコープ: 3件確定"） + 最新5件の詳細 |
| 41件以上 | カテゴリ別集約のみ + "詳細は draft を参照" |

```python
def format_confirmed_for_ai(confirmed: list, max_detail: int = 20) -> str:
    if len(confirmed) <= max_detail:
        return "\n".join(f"  - {item}" for item in confirmed)

    # カテゴリ別集約
    categories = {}
    for item in confirmed:
        # "質問: 回答" 形式から質問部分を抽出してカテゴリ推定
        cat = infer_category(item)
        categories.setdefault(cat, []).append(item)

    lines = []
    for cat, items in sorted(categories.items()):
        lines.append(f"  - {cat}: {len(items)}件確定")
    lines.append(f"  - 最新確定事項:")
    for item in confirmed[-5:]:
        lines.append(f"    - {item}")
    return "\n".join(lines)
```

## 6. 実装サブタスク分解

### 6.1 修正対象ファイル

| ファイル | 変更内容 | 行数目安 |
|---------|---------|---------|
| `internal/actions/templates/teraflow-req-agent.yml` | Python ロジック全面改修（パース追加、dialogue ノード改善、confirmed 更新） | ~150行変更 |
| `skills/discovery.yml` | システムプロンプトに再質問禁止ルール追加 | ~30行追加 |
| `internal/actions/templates_test.go` | パースロジックのテスト追加 | ~100行追加 |

### 6.2 サブタスク一覧

| # | タスク | 依存 | サイズ | 説明 |
|---|--------|------|--------|------|
| T1 | ユーザー回答パース関数の実装 | なし | M | `parse_user_answer` + `extract_questions_from_comment` を teraflow-req-agent.yml 内の Python に追加 |
| T2 | パースの tree 適用ロジック | T1 | S | パース結果で個別ノードを answered に更新する処理。mark_resolved 呼び出し前に Stage 1 パースを挿入 |
| T3 | confirmed 自動更新ロジック | T2 | S | `update_confirmed_from_parse` + `merge_ai_resolved` の二段階方式。summary.yaml 書き出し改修 |
| T4 | dialogue ノードの meta 拡張 | T2 | S | `parsed_answers` + `unparsed_portion` フィールドを dialogue ノードの meta に追加 |
| T5 | discovery.yml プロンプト改修 | なし | S | confirmed 再質問禁止ルール + unresolved 優先ルールをシステムプロンプトに追加 |
| T6 | テスト追加 | T1-T4 | M | パースロジックのユニットテスト（各回答パターン）+ 統合テスト（2ラウンド分のシミュレーション） |

**S = 1セッション, M = 2セッション**

### 6.3 推奨実装順序

```text
Phase A（パース基盤）:
  T1 → T2 → T3 → T4（順次。依存チェーンあり）

Phase B（AI制約。Phase Aと並行可能）:
  T5

Phase C（品質保証）:
  T6（Phase A完了後）
```

**最小リリース単位:** T1 + T2 + T3 で主要問題（P1, P2, P3）が全て解決する。T4, T5 は品質向上。T6 はリグレッション防止。

### 6.4 実装上の注意点

| 注意点 | 理由 |
|--------|------|
| teraflow-req-agent.yml 内の Python は**インデント厳守** | YAML heredoc 内のため、インデントずれで構文エラーになる |
| `parse_user_answer` は**既存の mark_resolved の前**に呼ぶ | Stage 1 パースが先、Stage 2 AI解析が後 |
| dialogue ノードの追加位置は**パース適用後** | parsed_answers を meta に記録するため |
| summary.yaml の confirmed は**重複排除**必須 | 同じ質問が複数ラウンドで確定される可能性 |
| `assistant_comment` が空の場合のフォールバック | 初回ラウンドや AI エラー時。fallback_comment からも質問抽出を試みる |
