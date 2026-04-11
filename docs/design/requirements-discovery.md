---
codd:
  node_id: "design:requirements-discovery"
  title: "grill-me 要件探索 × Discussion 統合アーキテクチャ設計書"
  depends_on:
    - id: "docs:concepts"
      relation: extends
    - id: "design:internal-packages"
      relation: extends
  tags:
    - grill-me
    - requirements
    - discovery
    - discussion
    - architecture
---

# grill-me 要件探索 × Discussion 統合アーキテクチャ設計書

## 1. grill-me アプローチ概要

### 1.1 grill-me とは

grill-me は、設計・計画の決定木の各分岐を 1 つずつ体系的に質問攻めにし、曖昧さや未定義を排除するアプローチである。teraflow ではこれを GitHub Discussion ベースの要件定義プロセスに適用する。

### 1.2 基本原則

| 原則 | 説明 |
|------|------|
| **決定木ベースの体系的質問** | 要件を階層的な決定木として構造化し、各分岐を順に解決する |
| **推奨回答付き** | 各質問にはAIの推奨回答を添える。ユーザーは「OK」で承認、「いいえ、〜」で修正 |
| **コードベース自動解決** | リポジトリの既存コード/設定から自動で解決可能な項目はユーザーに聞かずにAIが自動解決する |
| **1問ずつ or バッチ** | 初回はバッチ（全体像の把握）、2回目以降は1問ずつ深掘り |
| **累積的ドラフト** | 質問回答のたびに CoDD ドラフトを逐次更新し、到達点を可視化する |

### 1.3 既存フローとの差異

```text
既存フロー（req-agent 壁打ち）:
  Discussion コメント → 全コメント履歴取得 → LLM に全文渡す → 応答
  問題: コンテキスト肥大化、構造化されない自由対話、到達点が不明

grill-me フロー（本設計）:
  Discussion コメント → 状態ファイル（未解決分岐のみ）+ ドラフト読み込み
    → LLM に最小コンテキスト渡す → 次の質問生成 + ドラフト更新
  利点: コンテキスト一定、体系的、到達点が可視化
```

## 2. 状態管理設計

### 2.1 決定木スキーマ（YAML）

状態ファイルは Discussion ごとに 1 つ作成される。

```yaml
# .teraflow/discovery/{discussion-N}.yaml
version: "1"
discussion_number: 42
title: "ユーザー認証要件"
created_at: "2026-04-12T03:00:00Z"
updated_at: "2026-04-12T04:30:00Z"
mode: "sequential"  # "batch" | "sequential"

tree:
  - id: "scope"
    question: "この機能のスコープは何ですか？"
    category: "scope"
    status: "resolved"       # resolved | pending | skipped
    recommendation: "Webアプリのログイン機能に限定"
    answer: "Webアプリ + モバイルAPIの両方"
    resolved_at: "2026-04-12T03:15:00Z"
    resolved_by: "auto"      # "user" | "auto" | "ai-default"
    children:
      - id: "scope.web"
        question: "Web側の認証方式は？"
        category: "scope"
        status: "resolved"
        recommendation: "OAuth 2.0 + PKCE"
        answer: "OAuth 2.0 + PKCE で OK"
        resolved_at: "2026-04-12T03:20:00Z"
        resolved_by: "user"
        children: []
      - id: "scope.mobile"
        question: "モバイルAPI側の認証方式は？"
        category: "scope"
        status: "pending"
        recommendation: "JWT Bearer Token"
        answer: null
        children: []

  - id: "nfr.performance"
    question: "認証のレスポンスタイム要件は？"
    category: "non_functional"
    status: "pending"
    recommendation: "p95 < 500ms"
    answer: null
    children: []

  - id: "risk.session"
    question: "セッション管理のセキュリティ要件は？"
    category: "risk"
    status: "skipped"
    recommendation: null
    answer: null
    skip_reason: "Phase 2 で検討予定"
    children: []

summary:
  total: 12
  resolved: 5
  pending: 6
  skipped: 1
  progress_percent: 42
```

### 2.2 分岐ノードのフィールド定義

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `id` | string | Yes | ドット区切りの階層ID（例: `scope.web.oauth`） |
| `question` | string | Yes | ユーザーへの質問文 |
| `category` | string | Yes | カテゴリ（後述） |
| `status` | enum | Yes | `resolved` / `pending` / `skipped` |
| `recommendation` | string | No | AI の推奨回答 |
| `answer` | string | No | ユーザーの回答（resolved 時） |
| `resolved_at` | datetime | No | 解決日時 |
| `resolved_by` | enum | No | `user`（人間回答）/ `auto`（コードベース自動解決）/ `ai-default`（推奨回答承認） |
| `skip_reason` | string | No | スキップ理由（skipped 時） |
| `children` | array | No | 子ノード（深掘り質問） |

### 2.3 カテゴリ定義

| カテゴリ | 説明 | 典型的な質問例 |
|---------|------|--------------|
| `scope` | スコープ・境界 | 「この機能の対象ユーザーは？」 |
| `functional` | 機能要件 | 「ログイン失敗時の振る舞いは？」 |
| `non_functional` | 非機能要件 | 「レスポンスタイム要件は？」 |
| `acceptance` | 受入条件 | 「何を満たせばこの要件は完了？」 |
| `risk` | リスク・未確定事項 | 「セッション乗っ取り対策は？」 |
| `dependency` | 依存関係・前提条件 | 「既存の認証基盤との統合は？」 |
| `priority` | 優先度・Phase 分け | 「Phase 1 で実装する範囲は？」 |

## 3. CoDD ドラフト累積設計

### 3.1 ドラフトファイル

```text
.teraflow/discovery/{discussion-N}-draft.md
```

質問回答のたびに CoDD ドラフトを逐次更新する。ドラフトは最終的に `teraflow doc generate` の入力として活用される。

### 3.2 ドラフト構造

```markdown
---
codd:
  node_id: "req-user-auth"
  title: "ユーザー認証要件"
  depends_on: []
  status: "draft"
  source: "discussion:#42"
---

# ユーザー認証要件

> 進捗: 5/12 分岐解決済み (42%)
> 最終更新: 2026-04-12T04:30:00Z

## 背景・課題

（scope カテゴリの解決済み回答から自動構成）

## スコープ

- Webアプリ + モバイルAPIの両方を対象
- Web側: OAuth 2.0 + PKCE
- モバイルAPI側: **未確定**

## 機能要件

（functional カテゴリの解決済み回答から自動構成）

## 非機能要件

- レスポンスタイム: **未確定**

## 受入条件

（acceptance カテゴリの解決済み回答から自動構成）

## リスク・未確定事項

- セッション管理のセキュリティ要件: Phase 2 で検討予定（skipped）

## 依存関係・前提条件

（dependency カテゴリの解決済み回答から自動構成）
```

### 3.3 ドラフト更新ロジック

```text
ユーザー回答受信
    │
    ▼
[1] 状態ファイル更新（該当分岐を resolved に）
[2] ドラフト再生成:
    - resolved 分岐: 回答内容を該当セクションに反映
    - pending 分岐: 「**未確定**」マーカー
    - skipped 分岐: skip_reason を記載
    - 進捗バー更新
[3] ファイル書き出し + commit
```

ドラフトは「再生成」方式（差分パッチではなく全文再構成）。理由: 回答の追加が前のセクションに影響する場合がある（例: スコープ変更で機能要件も変わる）。

## 4. コンテキスト制御設計

### 4.1 LLM 入力の最適化

```text
LLM への入力（コンテキスト）:
┌─────────────────────────────────────────────┐
│ 1. システムプロンプト                        │  ~500 tok
│    grill-me ロール定義 + 出力形式指示        │
├─────────────────────────────────────────────┤
│ 2. 状態ファイル（未解決分岐のみ）            │  ~500-1500 tok
│    pending ノードの id/question/recommendation│
│    ※ resolved/skipped は含めない            │
├─────────────────────────────────────────────┤
│ 3. CoDD ドラフト（現在のスナップショット）   │  ~1000-3000 tok
│    ドラフト全文（到達点の把握用）            │
├─────────────────────────────────────────────┤
│ 4. 最新コメント（ユーザーの回答）            │  ~200-500 tok
│    Discussion の最新コメント1件のみ          │
├─────────────────────────────────────────────┤
│ 5. コードベースコンテキスト（自動解決用）    │  ~0-1000 tok
│    自動解決可能な項目がある場合のみ          │
└─────────────────────────────────────────────┘
合計: ~2200-6500 tok（入力側）
```

### 4.2 過去コメント全文を渡さない理由

| 現行（req-agent） | grill-me |
|-------------------|----------|
| コメント全文（last 20件） | 最新 1 件のみ |
| ~3000-8000 tok/回 | ~200-500 tok/回 |
| コメント増加でコスト増大 | **一定** |
| 文脈ロスト（LLM が全文を覚えきれない） | 状態ファイルが文脈を保持 |

状態ファイルとドラフトが「圧縮された記憶」として機能するため、過去コメント全文は不要。

### 4.3 トークン削減効果

```text
10 往復の壁打ちを想定:

現行（req-agent）:
  入力: 500 + 3000(履歴) + 500(最新) = 4000 tok/回
  10回: 4000 × 10 = 40,000 tok（履歴が累積するため実際はさらに増大）

grill-me:
  入力: 500 + 1000(未解決) + 2000(ドラフト) + 300(最新) = 3800 tok/回
  10回: 3800 × 10 = 38,000 tok（一定。未解決分岐は減少するためさらに低下）
```

### 4.4 LLM 出力形式

```json
{
  "action": "ask",
  "resolved_branches": [
    {
      "id": "scope.mobile",
      "answer": "JWT Bearer Token（ユーザー承認）",
      "resolved_by": "user"
    }
  ],
  "new_branches": [
    {
      "id": "scope.mobile.refresh",
      "parent_id": "scope.mobile",
      "question": "リフレッシュトークンの有効期限は？",
      "category": "functional",
      "recommendation": "30日。ローテーション方式。"
    }
  ],
  "auto_resolved": [
    {
      "id": "dependency.go_version",
      "answer": "Go 1.25.8 (go.mod から自動検出)",
      "resolved_by": "auto",
      "evidence": "go.mod: go 1.25.8"
    }
  ],
  "next_question": {
    "id": "scope.mobile",
    "question": "モバイルAPI側の認証方式は？",
    "recommendation": "JWT Bearer Token",
    "context": "Web側は OAuth 2.0 + PKCE に決定済みです。モバイル側も統一しますか？"
  },
  "draft_update": "（更新後の CoDD ドラフト全文）",
  "discussion_comment": "（Discussion に投稿するマークダウンコメント）"
}
```

## 5. ディレクトリ・ファイル構成

### 5.1 ディレクトリ名: `.teraflow/discovery/`

**推奨名称: `discovery`**（要件探索・発見を表す）

選定理由:

| 候補 | 評価 |
|------|------|
| `grill` | grill-me は手法名であり概念が伝わりにくい。× |
| `requirements-sessions` | 冗長。× |
| `dialogues` | 既存の req-agent の dialogue モードと混同。× |
| `inquiry` | 形式的すぎる。△ |
| **`discovery`** | 要件探索・発見のプロセスを端的に表現。**○** |
| `elicitation` | 要件抽出の正式用語だが長い。△ |

### 5.2 ファイル構成

```text
.teraflow/
├── index.yml                    # CoDD index（既存）
├── summaries/                   # AI 要約キャッシュ（既存）
├── graphrag/                    # GraphRAG ストレージ（Phase 2+）
└── discovery/                   # grill-me 要件探索セッション
    ├── discussion-42.yaml       # Discussion #42 の決定木状態
    ├── discussion-42-draft.md   # Discussion #42 の CoDD ドラフト
    ├── discussion-57.yaml       # Discussion #57 の決定木状態
    ├── discussion-57-draft.md   # Discussion #57 の CoDD ドラフト
    └── ...
```

### 5.3 ファイル命名規則

| ファイル | 命名パターン | 説明 |
|---------|-------------|------|
| 状態ファイル | `discussion-{N}.yaml` | Discussion 番号ベース |
| ドラフト | `discussion-{N}-draft.md` | CoDD frontmatter 付き |

### 5.4 .gitignore 考慮

`.teraflow/discovery/` は git 管理対象とする。理由:
- 状態ファイルとドラフトはプロジェクトの要件定義の記録として価値がある
- チームメンバーが探索の進捗を共有できる
- CI/CD で自動 commit/push する想定

## 6. ワークフロー連携設計

### 6.1 既存 req-agent フローの拡張方式

**方針: 新スキル `discovery` を追加し、既存 `requirements` スキルと共存させる。**

既存フローを壊さず、ラベルで grill-me モードを切り替える。

```text
Discussion コメント受信
    │
    ▼
[A] ラベル判定
    ├─ "discovery" or "要件探索" ラベルあり → grill-me モード（新フロー）
    └─ "requirements" ラベルのみ           → 既存壁打ちモード（変更なし）
```

### 6.2 新スキル: `skills/discovery.yml`

```yaml
name: discovery
version: "1"
description: "grill-me 方式の体系的要件探索"

trigger:
  labels: ["discovery", "要件探索", "grill"]
  categories: ["要件探索", "Discovery"]
  agent_types: ["requirements"]

prompts:
  system: |
    あなたは要件探索のスペシャリストです。grill-me アプローチに従い、
    決定木の各分岐を1つずつ体系的に質問し、要件を明確化します。

    ## ルール
    - 各質問にはあなたの推奨回答を添えること
    - コードベースで自動解決可能な項目は自動解決すること
    - 質問は依存関係を考慮した順序で行うこと（基盤→派生）
    - ユーザーが「OK」や推奨回答を承認した場合は resolved_by: "ai-default" とすること
    - ユーザーが「スキップ」と言った場合は status: "skipped" とし理由を記録すること
    - 全分岐が resolved/skipped になったら完了宣言すること

    ## 出力形式
    JSON のみ出力すること（マークダウンコードフェンスなし）。
    フィールド: action, resolved_branches, new_branches, auto_resolved,
    next_question, draft_update, discussion_comment

  init: |
    ユーザーが Discussion を新規作成しました。
    投稿内容から要件の決定木を初期生成してください。
    カテゴリ: scope, functional, non_functional, acceptance, risk, dependency, priority
    各カテゴリから 1-3 個の質問を生成し、最初のバッチ質問を出力してください。

  confirm: |
    ユーザーが要件の確定を宣言しました。
    CoDD ドラフトを最終版として整理し、doc generate の入力として使える形式で出力してください。

context:
  include:
    - "docs/requirements/*.md"
    - "docs/00_overview.md"
    - ".teraflow/discovery/"
  max_context_tokens: 6000

output:
  dialogue:
    header: "## 🔍 要件探索"
    footer: |
      > *teraflow discovery agent による要件探索です。*
      > *推奨回答で OK なら「OK」、修正がある場合はそのまま回答してください。*
      > *「スキップ」で後回しにできます。「要求確定」で最終要件を出力します。*
  confirm:
    header: "## 📋 要件確定"
    footer: "> *teraflow discovery agent による要件確定です。*"
  batch:
    header: "## 🔍 要件探索（バッチ）"
    footer: |
      > *番号で回答できます: 「1. OK 2. XX に変更 3. スキップ」*

options:
  max_tokens: 8192
  temperature: 0.3
```

### 6.3 ワークフロー改修: teraflow-req-agent.yml

既存のワークフローに discovery モード判定を追加:

```yaml
      - name: Determine response mode
        if: steps.check.outputs.has_key == 'true'
        id: mode
        run: |
          COMMENT="${{ github.event.comment.body }}"

          # Discussion ラベルに "discovery" があるか確認
          LABELS=$(gh api graphql -f query='
            query($id: ID!) {
              node(id: $id) {
                ... on Discussion {
                  labels(first: 10) { nodes { name } }
                }
              }
            }' -f id="${{ github.event.discussion.node_id }}" 2>/dev/null | \
            python3 -c "import json,sys; labels=[n['name'] for n in json.load(sys.stdin).get('data',{}).get('node',{}).get('labels',{}).get('nodes',[])]; print(','.join(labels))" \
            2>/dev/null || echo "")

          if echo "$LABELS" | grep -qiE "discovery|要件探索|grill"; then
            if echo "$COMMENT" | grep -q "要求確定"; then
              echo "mode=discovery-confirm" >> $GITHUB_OUTPUT
            else
              echo "mode=discovery" >> $GITHUB_OUTPUT
            fi
          elif echo "$COMMENT" | grep -q "要求確定"; then
            echo "mode=confirm" >> $GITHUB_OUTPUT
          else
            echo "mode=dialogue" >> $GITHUB_OUTPUT
          fi
```

### 6.4 discovery モード実行ステップ

```yaml
      - name: Discovery grill-me response
        if: startsWith(steps.mode.outputs.mode, 'discovery')
        id: discovery
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          DISC_NUM="${{ github.event.discussion.number }}"
          STATE_FILE=".teraflow/discovery/discussion-${DISC_NUM}.yaml"
          DRAFT_FILE=".teraflow/discovery/discussion-${DISC_NUM}-draft.md"

          # 状態ファイルがなければ init モード
          if [ ! -f "$STATE_FILE" ]; then
            SKILL_MODE="init"
          elif [ "${{ steps.mode.outputs.mode }}" = "discovery-confirm" ]; then
            SKILL_MODE="confirm"
          else
            SKILL_MODE="dialogue"
          fi

          # LLM 入力組み立て: 状態ファイル(未解決のみ) + ドラフト + 最新コメント
          python3 - << 'PYEOF' > /tmp/discovery_input.txt
          import yaml, sys, os

          disc_num = os.environ.get('DISC_NUM', '')
          state_path = f".teraflow/discovery/discussion-{disc_num}.yaml"
          draft_path = f".teraflow/discovery/discussion-{disc_num}-draft.md"

          input_parts = []

          # 状態ファイル（未解決分岐のみ抽出）
          if os.path.exists(state_path):
              with open(state_path) as f:
                  state = yaml.safe_load(f)
              def extract_pending(nodes, result=None):
                  if result is None: result = []
                  for node in (nodes or []):
                      if node.get('status') == 'pending':
                          result.append({
                              'id': node['id'],
                              'question': node['question'],
                              'category': node.get('category',''),
                              'recommendation': node.get('recommendation','')
                          })
                      extract_pending(node.get('children', []), result)
                  return result
              pending = extract_pending(state.get('tree', []))
              input_parts.append(f"## 未解決分岐 ({len(pending)}件)")
              input_parts.append(yaml.dump(pending, allow_unicode=True, default_flow_style=False))
              summary = state.get('summary', {})
              input_parts.append(f"進捗: {summary.get('resolved',0)}/{summary.get('total',0)}")

          # ドラフト
          if os.path.exists(draft_path):
              with open(draft_path) as f:
                  input_parts.append("## 現在の CoDD ドラフト")
                  input_parts.append(f.read())

          print('\n'.join(input_parts))
          PYEOF

          # 最新コメント追加
          echo "" >> /tmp/discovery_input.txt
          echo "## 最新コメント" >> /tmp/discovery_input.txt
          echo "${{ github.event.comment.body }}" >> /tmp/discovery_input.txt

          teraflow agent assign \
            --type requirements \
            --skill discovery \
            --input-file /tmp/discovery_input.txt \
            --format json \
            --config .github/teraflow.yml \
            > /tmp/discovery_result.json 2>/tmp/discovery_err.log || true

          # 結果パース + 状態ファイル/ドラフト更新
          python3 - << 'PYEOF'
          import json, yaml, os, sys
          from datetime import datetime, timezone

          disc_num = os.environ.get('DISC_NUM', '')
          result_path = '/tmp/discovery_result.json'
          state_path = f".teraflow/discovery/discussion-{disc_num}.yaml"
          draft_path = f".teraflow/discovery/discussion-{disc_num}-draft.md"

          os.makedirs('.teraflow/discovery', exist_ok=True)

          with open(result_path) as f:
              raw = json.load(f)
          output = raw.get('output', '')

          # JSON パース（LLM 出力）
          try:
              data = json.loads(output)
          except json.JSONDecodeError:
              data = {'discussion_comment': output}

          # 状態ファイル更新
          if os.path.exists(state_path):
              with open(state_path) as f:
                  state = yaml.safe_load(f)
          else:
              state = {
                  'version': '1',
                  'discussion_number': int(disc_num),
                  'created_at': datetime.now(timezone.utc).isoformat(),
                  'mode': 'batch',
                  'tree': [],
                  'summary': {'total': 0, 'resolved': 0, 'pending': 0, 'skipped': 0}
              }

          # resolved_branches を反映
          # new_branches を追加
          # auto_resolved を反映
          # summary 再計算
          state['updated_at'] = datetime.now(timezone.utc).isoformat()

          with open(state_path, 'w') as f:
              yaml.dump(state, f, allow_unicode=True, default_flow_style=False)

          # ドラフト更新
          if 'draft_update' in data and data['draft_update']:
              with open(draft_path, 'w') as f:
                  f.write(data['draft_update'])

          # Discussion コメント用出力
          comment = data.get('discussion_comment', output)
          with open('/tmp/discovery_comment.txt', 'w') as f:
              f.write(comment)
          PYEOF

          # git commit + push
          git add .teraflow/discovery/ || true
          git diff --cached --quiet || \
            git commit -m "discovery: update session for discussion #${DISC_NUM}" && \
            git push origin HEAD || true
```

## 7. 決定木テンプレート設計

### 7.1 要件カテゴリ別テンプレート

決定木の初期生成時に使用するテンプレート。ユーザーの最初の投稿内容に応じて、該当するカテゴリの質問を自動選択する。

**スコープ（scope）**:
```yaml
- id: "scope.target_users"
  question: "この機能の対象ユーザーは誰ですか？"
  recommendation: null  # 投稿内容から推定
- id: "scope.boundary"
  question: "この機能のスコープ外は何ですか？（明示的に除外するもの）"
  recommendation: null
- id: "scope.platform"
  question: "対象プラットフォームは？（Web/Mobile/API/CLI等）"
  recommendation: null  # コードベースから自動検出を試みる
```

**機能要件（functional）**:
```yaml
- id: "func.happy_path"
  question: "正常系のメインフローを教えてください"
  recommendation: null
- id: "func.error_handling"
  question: "エラー時の振る舞いは？（バリデーション失敗、外部API障害等）"
  recommendation: "エラーメッセージを表示し、入力を保持してリトライ可能にする"
- id: "func.edge_cases"
  question: "考慮すべきエッジケースはありますか？"
  recommendation: null
```

**非機能要件（non_functional）**:
```yaml
- id: "nfr.performance"
  question: "レスポンスタイム要件は？"
  recommendation: "p95 < 500ms"
- id: "nfr.availability"
  question: "可用性要件は？（SLA等）"
  recommendation: "99.9%"
- id: "nfr.scalability"
  question: "想定ユーザー数/トラフィック量は？"
  recommendation: null
```

**受入条件（acceptance）**:
```yaml
- id: "ac.done_definition"
  question: "この要件の「完了」の定義は？"
  recommendation: null
- id: "ac.test_scenarios"
  question: "必須のテストシナリオは？"
  recommendation: "正常系E2E + 異常系バリデーション + パフォーマンステスト"
```

**リスク（risk）**:
```yaml
- id: "risk.security"
  question: "セキュリティ上の懸念事項は？"
  recommendation: null  # コードベースの既存セキュリティパターンから推定
- id: "risk.data_migration"
  question: "既存データへの影響はありますか？"
  recommendation: null
- id: "risk.rollback"
  question: "ロールバック計画は必要ですか？"
  recommendation: "Feature flag で段階的リリース"
```

### 7.2 決定木の初期生成フロー

```text
Discussion 新規作成（discovery ラベル付き）
    │
    ▼
[1] Discussion 本文を LLM に渡す
    プロンプト: init モード（skills/discovery.yml prompts.init）
    │
    ▼
[2] LLM が本文を分析:
    - 言及されているカテゴリを特定
    - 各カテゴリからテンプレート質問を選択（1-3個/カテゴリ）
    - 本文で既に明確な項目は auto_resolved（自動解決）
    - 推奨回答を投稿内容から推定
    │
    ▼
[3] 初期決定木を生成（10-20 分岐）
    │
    ▼
[4] 初回バッチ質問を Discussion にコメント:

    ## 🔍 要件探索（バッチ）

    投稿内容を分析しました。以下の質問に回答してください。

    ### スコープ
    1. **対象ユーザーは？** → 推奨: 管理者ユーザー
    2. **スコープ外は？** → 推奨: モバイル対応は Phase 2

    ### 機能要件
    3. **正常系フローは？** → 推奨: （投稿から推定した内容）
    4. **エラー時の振る舞いは？** → 推奨: エラーメッセージ表示

    ### 非機能要件
    5. **レスポンスタイムは？** → 推奨: p95 < 500ms

    > 番号で回答できます: 「1. OK 2. XX に変更 3. スキップ」
    > 「OK」は推奨回答の承認です。
```

### 7.3 質問順序の依存関係制御

```text
質問順序の原則:
  1. scope → functional → non_functional → acceptance → risk → dependency → priority
  2. 同カテゴリ内: 基盤的な質問（親ノード）→ 派生的な質問（子ノード）
  3. 子ノードは親ノードが resolved になるまで pending のまま保留

例:
  scope.platform = "Web + Mobile"  (resolved)
    ├─ scope.web.auth → pending → LLM が次に質問
    └─ scope.mobile.auth → pending → scope.web.auth の後に質問
```

## 8. 要求確定への接続

### 8.1 grill 完了判定

```go
// internal/discovery/state.go

// IsComplete は全分岐が resolved または skipped かを判定
func (s *SessionState) IsComplete() bool {
    return s.Summary.Pending == 0
}

// CompletionReport は完了レポートを生成
func (s *SessionState) CompletionReport() string {
    return fmt.Sprintf(
        "要件探索完了: %d/%d 解決済み, %d スキップ, 進捗 %d%%",
        s.Summary.Resolved, s.Summary.Total,
        s.Summary.Skipped, s.Summary.ProgressPercent,
    )
}
```

全分岐が resolved/skipped になった場合、自動的に完了メッセージを Discussion に投稿:

```text
## ✅ 要件探索完了

全 12 分岐の探索が完了しました。
- 解決済み: 10
- スキップ: 2

「要求確定」と発言すると、探索結果から CoDD 要件定義書を生成します。
```

### 8.2 CoDD ドラフト → doc generate 接続

```text
「要求確定」発言
    │
    ▼
[1] discovery-confirm モード発動
[2] CoDD ドラフト（.teraflow/discovery/discussion-N-draft.md）を読み込み
[3] ドラフトを最終整理（LLM で推敲）
[4] 整理済みドラフトを docs/ に配置
    ├─ 方式A: doc generate に渡す（Discussion 全文の代わりにドラフトを入力）
    └─ 方式B: ドラフトをそのまま docs/requirements/ にコピー
[5] index.yml 更新
[6] PR 作成
[7] Discussion にリンクコメント投稿
```

**方式A（推奨）**: `doc generate` の入力として既存のドラフトを活用。`generator.go` の `structurize()` をスキップし、ドラフトの frontmatter をそのまま使用する。これにより、探索中に蓄積された構造がそのまま最終文書に反映される。

```go
// cmd/doc.go に --from-draft フラグ追加
// teraflow doc generate --from-draft .teraflow/discovery/discussion-42-draft.md
//
// ドラフトから直接 CoDD 文書を生成（Discussion fetch + AI structurize をスキップ）
```

### 8.3 skipped 分岐の扱い

skipped 分岐は CoDD ドラフトの「リスク・未確定事項」セクションに記載される。これにより、後続の設計フェーズで意識的に対処できる。

## 9. 巨大プロジェクト対応

### 9.1 決定木が 100+ 分岐になった場合

**スケーラビリティ設計**:

| 対策 | 説明 |
|------|------|
| **セクション分割** | 決定木を複数のサブツリーに分割。各サブツリーを独立したセッションとして管理。 |
| **自動折りたたみ** | 深度 3 以上の resolved 分岐は状態ファイルで折りたたみ（`collapsed: true`）。LLM には渡さない。 |
| **優先度ベースフィルタ** | pending 分岐が 20 以上の場合、カテゴリ優先度順に上位 10 件のみ LLM に渡す。 |
| **バッチ上限** | 1 回のバッチ質問は最大 7 問。認知負荷を考慮。 |

### 9.2 状態ファイルの折りたたみ

```yaml
tree:
  - id: "scope"
    status: "resolved"
    collapsed: true       # ← 深度0のノードが全子ノード resolved なら折りたたみ
    children_summary:     # ← 折りたたみ時の要約
      total: 5
      resolved: 5
      key_decisions:
        - "対象: Web + Mobile"
        - "認証: OAuth 2.0 + PKCE"
    children: [...]       # ← ファイルには残るが LLM には渡さない
```

### 9.3 Discussion が長大になった場合

Discussion 自体の長さは grill-me モードでは問題にならない（過去コメント全文を渡さないため）。ただし、以下の対策を設ける:

| 問題 | 対策 |
|------|------|
| Discussion の可読性低下 | 定期的に「探索サマリー」コメントを投稿（10 問解決ごと） |
| 状態ファイルの肥大化 | 折りたたみ + YAMLアンカー（`&`/`*`）の活用 |
| ドラフトの肥大化 | ドラフトはセクション単位で管理。最大 10,000 トークンを目安に分割 |

### 9.4 複数 Discussion の並行探索

異なる機能要件を複数の Discussion で並行して探索できる。各 Discussion は独立した状態ファイル + ドラフトを持つため、干渉しない。

## 10. 段階的導入計画

### Phase 1: 基盤（状態管理 + スキル定義）

**実装内容**:
- `internal/discovery/` パッケージ新規作成
  - `state.go`: SessionState 型、決定木操作、進捗計算
  - `draft.go`: CoDD ドラフト生成・更新
  - テスト
- `skills/discovery.yml` 新規作成
- `.teraflow/discovery/` ディレクトリ管理

**価値**: grill-me の核となるデータモデルの確立

**後方互換性**: 既存の req-agent フローに影響なし（新ラベル `discovery` でのみ発動）

**見積り**: M 工数、サブタスク 2-3 件

### Phase 2: ワークフロー統合

**実装内容**:
- `teraflow-req-agent.yml` に discovery モード追加
- ワークフロー内の状態ファイル/ドラフト読み書きロジック
- Discussion コメント投稿（バッチ質問/個別質問/サマリー）
- `cmd/discovery.go`: `teraflow discovery status <discussion-N>` コマンド

**価値**: GitHub Discussion 上で grill-me ワークフローが動作

**後方互換性**: ラベルベース切り替え。`requirements` ラベルの Discussion は従来通り動作

**見積り**: L 工数、サブタスク 3-4 件

### Phase 3: 確定フロー + doc generate 連携

**実装内容**:
- `doc generate --from-draft` フラグ追加
- discovery-confirm モード: ドラフト→最終文書→PR
- 自動完了判定 + 完了メッセージ
- skipped 分岐のリスクセクション反映

**価値**: 探索結果が直接 CoDD 文書として確定

**後方互換性**: `doc generate --discussion` は従来通り動作。`--from-draft` は追加オプション

**見積り**: M 工数、サブタスク 2-3 件

### Phase 4: 高度機能

**実装内容**:
- コードベース自動解決（go.mod/pyproject.toml/既存設計文書からの自動回答）
- 決定木テンプレートのカスタマイズ（teraflow.yml 設定）
- 折りたたみ/バッチ上限/優先度フィルタ（巨大プロジェクト対応）
- GraphRAG 連携（既存文書から依存関係を自動推定）

**価値**: 自動化度の向上、大規模対応

**見積り**: L 工数、サブタスク 4-5 件

### 各 Phase の独立価値

```text
Phase 1 だけでも: 状態管理の型が確立 → 手動で grill-me を実践可能
Phase 1+2 で:     Discussion 上で自動 grill-me → 要件探索の効率化
Phase 1+2+3 で:   探索→確定→CoDD の一気通貫フロー
Phase 1+2+3+4 で: 自動解決+大規模対応 → エンタープライズ対応
```
