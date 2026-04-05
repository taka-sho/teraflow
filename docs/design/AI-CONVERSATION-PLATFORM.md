---
codd:
  node_id: "design:ai-conversation-platform"
  title: "AI対話基盤 Phase 1-5 計画書"
  depends_on:
    - id: "design:system-overview"
      relation: extends
    - id: "adr:003-ai-integration"
      relation: derives_from
    - id: "design:agent-provider"
      relation: extends
    - id: "docs:roadmap"
      relation: implements
---

# AI対話基盤 Phase 1-5 計画書

## 1. ビジョン・目的

### 1.1 統合コンセプト: 共通フレーム x CoDD x 生成AI

teraflowは「共通フレーム（SLCP-JCF）のプロセスに沿って、AIと対話しながらソフトウェアを開発する」基盤を目指す。

```
共通フレーム          CoDD              生成AI
(プロセス定義)    (依存関係管理)      (対話・生成)
     |                 |                  |
     v                 v                  v
  ┌────────────────────────────────────────┐
  │           teraflow 統合基盤            │
  │                                        │
  │  要件定義 → 設計 → 実装 → テスト       │
  │     ↑         ↑        ↑       ↑      │
  │    AI壁打ち  AI壁打ち  AIレビュー  AI分析 │
  │     ↓         ↓        ↓       ↓      │
  │  CoDD文書   CoDD文書  PR/Issue  レポート │
  └────────────────────────────────────────┘
```

核心的な前提:

- **対話フェーズ**: Discussion/Issue/PR上でAIと壁打ちし、人間が「確定」を宣言する
- **文書化フェーズ**: 確定した内容をCoDD文書（frontmatter付きmarkdown）として自動生成
- **コンテキスト認識**: AIはdocs/配下の既存文書+実装コードを考慮して壁打ちに参加

### 1.2 現状と課題

現在のAIエージェント（`internal/agent/`）は単発リクエスト-レスポンス型:

- `teraflow agent assign --type requirements --input-file input.txt`
- 会話履歴なし、コンテキスト制限なし、確定トリガーなし
- 7種のエージェント型（requirements/review/implement/ci-fix/conflict/incident/maintenance）
- プロバイダー: Anthropic（デフォルト）、OpenAI、Claude Code、Custom

課題:

1. **対話性の欠如**: 1回きりの応答で壁打ちができない
2. **コンテキスト不足**: docs/配下の既存文書を参照しない
3. **確定プロセスなし**: いつ要件が確定したか機械的に判定できない
4. **文書自動生成なし**: 確定後のCoDD文書作成は手動

## 2. 全体アーキテクチャ

### 2.1 4層アーキテクチャ

```
┌─────────────────────────────────────────────────────────────┐
│                     通信層 (Phase 5+)                        │
│  MCP (Model Context Protocol) — 将来の外部ツール連携パス     │
├─────────────────────────────────────────────────────────────┤
│                     AI能力層 (Phase 2)                       │
│  Skills: 場面ごとの参照戦略 + プロンプト + 出力形式          │
│  ┌─────────────┐ ┌──────────────┐ ┌─────────────┐          │
│  │ req-skill   │ │ design-skill │ │ review-skill│ ...      │
│  │ (要件議論)  │ │ (設計議論)   │ │ (レビュー)  │          │
│  └─────────────┘ └──────────────┘ └─────────────┘          │
├─────────────────────────────────────────────────────────────┤
│                     実行制御層 (Phase 4)                     │
│  Hooks: 自動維持タイミング制御                               │
│  ┌───────────────────────────────────────────────────┐      │
│  │ teraflow.yml hooks:                                │      │
│  │   on_discussion_comment: → skill選択 → AI応答      │      │
│  │   on_confirmation:       → CoDD文書生成            │      │
│  │   on_pr_opened:          → review-skill起動        │      │
│  └───────────────────────────────────────────────────┘      │
├─────────────────────────────────────────────────────────────┤
│                     データ層 (Phase 3)                       │
│  Index + 要約キャッシュ — コンテキスト量制御                 │
│  ┌────────────────┐  ┌───────────────────┐                  │
│  │ .teraflow/     │  │ .teraflow/        │                  │
│  │  index.yml     │  │  summaries/       │                  │
│  │ (文書メタ情報) │  │  (要約キャッシュ)  │                  │
│  └────────────────┘  └───────────────────┘                  │
├─────────────────────────────────────────────────────────────┤
│                     基盤層 (Phase 1)                         │
│  既存: agent.go, provider.go, GitHub Actions workflows      │
│  改修: プロンプト改修, 会話履歴取得, 確定トリガー            │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 データフロー概要

```
Discussion作成
    │
    ├─ [Phase 1] req-agent が対話モードで応答
    │     └─ 会話履歴を取得し、文脈を維持
    │
    ├─ [Phase 2] Skill自動選択（ラベル/カテゴリから判定）
    │     └─ skills/req.yml → 参照戦略 + プロンプト + 出力形式
    │
    ├─ [Phase 3] Index + 要約でコンテキスト構築
    │     └─ docs/配下の関連文書を要約付きで注入
    │
    ├─ [Phase 4] Hooks が自動でSkill起動
    │     └─ teraflow.yml hooks: on_discussion_comment
    │
    └─ [Phase 5] 「確定」トリガー → CoDD文書自動生成
          └─ teraflow doc generate → docs/requirements/req-XXX.md
```

## 3. フェーズ詳細

### 3.1 Phase 1: 対話化

**目的**: req-agentを「1回きり応答」から「壁打ち対話」に進化させる

**対応内容**:

| 項目 | 詳細 |
|------|------|
| プロンプト改修 | GetSystemPrompt()を対話向けに改修。前回コメント参照、質問返し、構造化提案を含む |
| 会話履歴取得 | GitHub GraphQL APIでDiscussion/Issueの全コメントを時系列取得 |
| 確定トリガー | teraflow.yml の confirmation.trigger (現在: "確定") を検出し、最終要件サマリーを出力 |
| ワークフロー改修 | teraflow-req-agent.yml: discussion_comment イベントで過去コメントも渡す |

**成果物**:

- `internal/agent/agent.go`: GetSystemPrompt() 改修（対話型プロンプト）
- `internal/agent/conversation.go`: 会話履歴取得ロジック（新規）
- `internal/actions/templates/teraflow-req-agent.yml`: 会話履歴をinputに含める改修
- `cmd/agent.go`: `--conversation-id` フラグ追加（Discussion ID指定）

**前提条件**:

- GitHub GraphQL API でDiscussionコメントを取得できること
- GITHUB_TOKEN に discussions:read 権限があること

**依存関係**: なし（現行コードベースで開始可能）

### 3.2 Phase 2: Skills導入

**目的**: 場面ごとに最適な参照戦略・プロンプト・出力形式を定義する仕組みを導入

**対応内容**:

| 項目 | 詳細 |
|------|------|
| skills/*.yml定義 | 各場面の設定をYAMLで外部化。system_prompt, context_strategy, output_format を含む |
| Skill自動選択 | Discussion/Issueのラベル or カテゴリ名からSkillを自動判定 |
| 多Skill実装 | 要件議論、設計議論、コードレビュー、バグ対応、変更要求の5場面 |

**Skill定義フォーマット**:

```yaml
# skills/req.yml
name: requirements
trigger:
  labels: ["requirements", "要件"]
  categories: ["要件議論"]
system_prompt: |
  あなたはソフトウェア要件整理の専門家です。
  ユーザーと対話しながら要件を明確化してください。
  docs/配下の関連文書を参照し、既存要件との整合性を確認してください。
context_strategy:
  include:
    - "docs/requirements/*.md"
    - "docs/guide/concepts.md"
  max_tokens: 3000
output_format:
  type: markdown
  template: |
    ## 要件整理

    ### 機能要件
    {functional_requirements}

    ### 非機能要件
    {non_functional_requirements}

    ### 未確定事項
    {open_questions}
```

**成果物**:

- `skills/` ディレクトリ: req.yml, design.yml, review.yml, bugfix.yml, change-request.yml
- `internal/skill/loader.go`: Skill定義の読み込み・バリデーション
- `internal/skill/selector.go`: ラベル/カテゴリからSkill自動選択
- `cmd/skill.go`: `teraflow skill list` / `teraflow skill show <name>`

**前提条件**: Phase 1 完了（対話型プロンプト基盤）

**依存関係**: Phase 1

### 3.3 Phase 3: Index + 要約キャッシュ

**目的**: docs/配下の文書をAIが効率的に参照できるようにする（コンテキスト量制御）

**対応内容**:

| 項目 | 詳細 |
|------|------|
| .teraflow/index.yml | docs/配下の全CoDD文書のメタ情報（node_id, title, depends_on, 更新日時）を集約 |
| .teraflow/summaries/ | 各文書の要約キャッシュ（AI生成、200-300 tokens/文書） |
| コンテキスト構築 | Skill の context_strategy に基づき、関連文書の要約+全文を選択的に注入 |
| 大規模対応 | 文書数が増えても一定のトークン数内に収まるよう、要約レベルを動的調整 |

**コンテキスト量制御設計（Phase 3の核心）**:

```
トークン配分（1リクエストあたり上限 ~6,000 tokens）:
┌──────────────────────────────────────────┐
│ system prompt          ~500 tok          │
│ 会話履歴（直近N件）    ~2,000 tok         │
│ 関連文書要約           ~1,000 tok         │
│ 関連文書全文（最重要1件）~2,000 tok        │
│ ユーザー入力           ~500 tok           │
│────────────────────────────────────────── │
│ 合計                   ~6,000 tok         │
└──────────────────────────────────────────┘

コンテキスト選択アルゴリズム:
1. Skill の context_strategy.include パターンにマッチする文書を列挙
2. CoDD depends_on 関係で距離が近い文書を優先
3. 会話中のキーワードと文書titleの類似度でランキング
4. トークン上限内に収まるよう、要約→全文の順で注入
```

**成果物**:

- `internal/index/builder.go`: index.yml 生成・更新
- `internal/index/summarizer.go`: AI要約生成・キャッシュ管理
- `internal/context/assembler.go`: コンテキスト構築（トークン配分制御）
- `cmd/index.go`: `teraflow index build` / `teraflow index status`

**前提条件**: Phase 2 完了（Skillのcontext_strategyを使用）

**依存関係**: Phase 2

### 3.4 Phase 4: Hooks導入

**目的**: AIの自動起動タイミングをteraflow.ymlで宣言的に制御する

**対応内容**:

| 項目 | 詳細 |
|------|------|
| hooks:セクション | teraflow.yml にイベント駆動の自動処理を定義 |
| 自動維持 | Index/要約キャッシュの自動更新タイミング制御 |
| GitHub Actions連携 | Hookの実行をGitHub Actionsワークフローに変換 |

**teraflow.yml hooks設計**:

```yaml
hooks:
  on_discussion_created:
    - skill: auto        # ラベル/カテゴリから自動選択
      action: respond    # AI応答を投稿

  on_discussion_comment:
    - skill: auto
      action: respond
      condition:
        not_author: "github-actions[bot]"  # 自分のコメントに反応しない

  on_confirmation:       # confirmation.trigger を検出
    - action: summarize  # 最終要件サマリー生成
    - action: generate   # CoDD文書自動生成（Phase 5）

  on_push:
    - action: index_update  # docs/変更時にindex.yml更新
      paths: ["docs/**"]

  on_pr_opened:
    - skill: review
      action: respond
```

**成果物**:

- `internal/hooks/parser.go`: hooks:セクションのパース
- `internal/hooks/executor.go`: Hook実行エンジン
- `internal/actions/generator.go`: HookからGitHub Actions workflow生成
- teraflow.yml スキーマ拡張

**前提条件**: Phase 3 完了（IndexをHookで自動維持）

**依存関係**: Phase 3

### 3.5 Phase 5: CoDD文書自動生成 + トレーサビリティ

**目的**: 確定した対話内容からCoDD文書を自動生成し、トレーサビリティを確保する

**対応内容**:

| 項目 | 詳細 |
|------|------|
| teraflow doc generate | 確定済みDiscussionからCoDD文書を自動生成 |
| frontmatter自動付与 | node_id, title, depends_on を対話内容+既存文書から推定 |
| teraflow trace | 要件→設計→実装→テストのトレーサビリティチェーン表示 |
| Discussion-文書リンク | 生成文書にソースDiscussionへのリンクを自動記載 |

**文書生成フロー**:

```
Discussion #42 で「確定」トリガー検出
    │
    ├─ 1. 全コメント取得
    ├─ 2. AI が要件を構造化
    ├─ 3. CoDD frontmatter 生成
    │     ├─ node_id: 会話内容から推定 (例: "req:user-auth")
    │     ├─ title: 会話のタイトルから生成
    │     └─ depends_on: 既存文書との関連を推定
    ├─ 4. markdown本文生成
    ├─ 5. docs/requirements/req-user-auth.md に書き出し
    ├─ 6. index.yml 更新
    └─ 7. PR作成（人間レビュー用）
```

**成果物**:

- `cmd/doc.go`: `teraflow doc generate --discussion <id>` / `teraflow doc list`
- `cmd/trace.go`: `teraflow trace <node_id>` — トレーサビリティチェーン表示
- `internal/doc/generator.go`: CoDD文書生成エンジン
- `internal/trace/resolver.go`: depends_on グラフ走査

**前提条件**: Phase 4 完了（Hooksによる自動起動基盤）

**依存関係**: Phase 4

## 4. 場面別AI役割表

| 場面 | Skill | トリガー | AIの役割 | 出力形式 | Phase |
|------|-------|---------|---------|---------|-------|
| 要件議論 | req | Discussion (要件ラベル) | 要件の明確化・構造化・質問 | 機能/非機能要件リスト | 1-2 |
| 設計議論 | design | Discussion (設計ラベル) | 設計提案・トレードオフ分析 | 設計案+ADR形式 | 2 |
| コードレビュー | review | PR opened/synchronize | セキュリティ/品質/設計の指摘 | severity付き指摘リスト | 1 |
| バグ対応 | bugfix | Issue (bugラベル) | 原因調査・修正提案 | 5W1H分析+修正案 | 2 |
| 変更要求 | change-request | Issue (enhancementラベル) | 影響範囲分析・工数見積もり | CoDD影響グラフ+見積もり | 3+ |

**場面ごとのコンテキスト参照戦略**:

```
要件議論:
  参照: docs/requirements/*.md, docs/guide/concepts.md
  理由: 既存要件との整合性確認、重複検出

設計議論:
  参照: docs/design/*.md, docs/adr/*.md
  理由: 既存設計との一貫性、ADR判断履歴の参照

コードレビュー:
  参照: docs/design/internal-packages.md, docs/design/cli-interface.md
  理由: 設計書と実装の乖離検出

バグ対応:
  参照: .teraflow/incident-log.yml, docs/design/*.md
  理由: 過去のインシデントパターン、設計意図の確認

変更要求:
  参照: .teraflow/index.yml (CoDD全体), ROADMAP.md
  理由: 影響範囲のグラフ分析、ロードマップとの整合
```

## 5. コンテキスト量制御設計

### 5.1 トークン配分モデル

AIプロバイダーのコンテキストウィンドウを効率的に使用するため、固定枠方式を採用する。

```
入力トークン配分（合計上限: ~6,000 tokens）:

  ┌─────────────────────────────┐
  │ system prompt       ~500    │ ← Skill定義から生成
  │                             │
  │ 会話履歴（直近N件） ~2,000   │ ← 最新から逆順に格納
  │                             │    古いコメントは要約化
  │ 関連文書要約        ~1,000   │ ← index.yml から選択
  │                             │    summaries/ から読み込み
  │ 関連文書全文        ~2,000   │ ← 最関連1件のみ全文
  │                             │    トークン超過時は切り詰め
  │ ユーザー入力         ~500    │ ← 最新コメント
  └─────────────────────────────┘
```

### 5.2 コンテキスト選択アルゴリズム

```
入力: Skill.context_strategy, 会話履歴, ユーザー入力
出力: context_window (構築済みプロンプト)

Step 1: 文書候補列挙
  → Skill.context_strategy.include のglobパターンでマッチ

Step 2: 関連度スコアリング
  → CoDD depends_on の距離（近い = 高スコア）
  → ユーザー入力とのキーワード重複度
  → 最終更新日時（新しい = 高スコア）

Step 3: トークン配分
  → スコア上位N件の要約を ~1,000 tok 枠に格納
  → スコア最上位1件の全文を ~2,000 tok 枠に格納
  → 枠超過時は末尾切り詰め

Step 4: 会話履歴圧縮
  → 直近3件はそのまま格納
  → 4件目以降は要約化（1コメント → 1行サマリー）
  → ~2,000 tok 枠に収まるまで古い方から削除
```

### 5.3 要約キャッシュ管理

```
.teraflow/
  index.yml              # 全CoDD文書のメタ情報
  summaries/
    req-slcp-jcf.txt     # docs/requirements/req-slcp-jcf-compliance.md の要約
    design-rbac.txt      # docs/design/rbac-gate-design.md の要約
    ...

キャッシュ更新タイミング:
  - teraflow index build 実行時
  - Phase 4以降: on_push hook で docs/ 変更検出時に自動更新

キャッシュ無効化:
  - 元文書の更新日時 > 要約の更新日時 → 再生成
  - index.yml のハッシュ比較で差分検出
```

## 6. Phase間の依存関係

```
Phase 1: 対話化
  │
  └─→ Phase 2: Skills導入
        │
        └─→ Phase 3: Index + 要約キャッシュ
              │
              └─→ Phase 4: Hooks導入
                    │
                    └─→ Phase 5: CoDD文書自動生成
```

各Phaseは前Phaseの成果物を前提とするが、Phase 1 は現行コードベースで即座に開始可能。

## 7. 実装優先度とROADMAP対応

| Phase | ROADMAP | 優先度 | 見積もり |
|-------|---------|--------|---------|
| Phase 1 | v0.2.x (In Progress) | 最優先 | M (cmd_129で並行実装中) |
| Phase 2 | v0.3.x (Near-term) | 高 | L |
| Phase 3 | v0.3.x (Near-term) | 中 | L |
| Phase 4 | v1.0+ (Future) | 中 | L |
| Phase 5 | v1.0+ (Future) | 低 | XL |
