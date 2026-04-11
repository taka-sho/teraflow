# V字モデルパイプライン利用ガイド

teraflow v0.5.0 で導入された V字モデルパイプラインの利用ガイドです。
Discussion での要件定義から、設計・実装・検証までの一気通貫フローを解説します。

技術的な設計詳細は [V字モデル設計書](../design/v-model-pipeline.md) を参照してください。


## 1. クイックスタート

「認証機能」を例に、最小手順で Discussion から実装までの流れを体験します。

### Step 1: Discussion 作成（要件定義）

GitHub リポジトリで Discussion を新規作成し、`discovery` ラベルを付与します。

```
Title: 認証機能の追加
Body: ユーザー認証機能を追加したい。OAuth2 対応。
Labels: discovery
```

grill-me エージェントが自動で質問を開始します。推奨回答が提示されるので、「OK」で承認するか、修正して回答してください。

### Step 2: 進捗確認

```bash
teraflow discovery status discussion-42
```

期待される結果:
```
Discussion: #42
Title: 認証機能の追加
Progress: 8/12 (66%)
Pending: 4
Skipped: 0
Blocked: 0
```

### Step 3: 要件確定

全質問に回答したら、Discussion で「要求確定」とコメントします。CoDD 要件文書の PR が自動作成されます。レビューして承認・マージしてください。

### Step 4: 設計フェーズへ遷移

Issue を作成し、`phase: basic-design` ラベルを付与します。

```bash
# Wave 計画を確認
teraflow plan waves --phase basic-design

# 整合性チェック
teraflow validate --phase basic-design
```

### Step 5: 設計書の自動生成

```bash
teraflow generate --phase basic-design
```

Wave 1〜5 で設計書が順次生成され、PR が作成されます。

### Step 6: 実装

設計書マージ後、`phase: implementation` ラベルの Issue を作成します。

```bash
teraflow implement --design docs/design/auth-api.md
```

modules 単位で並列にコード生成され、PR が作成されます。

### Step 7: 検証

V字右側のテストが自動生成・実行されます。


## 2. 要件定義フェーズ

### 2.1 Discussion の作成

GitHub Discussion で要件を議論します。2つのモードがあります。

| ラベル | モード | 説明 |
|--------|--------|------|
| `requirements` | 従来の壁打ちモード | 自由対話形式で要件を議論 |
| `discovery` | grill-me モード | 決定木ベースの体系的質問 |

> **推奨**: `discovery` ラベルを使った grill-me モードは、体系的に要件を漏れなく洗い出せます。

### 2.2 grill-me による質問応答

`discovery` ラベルを付けると、grill-me エージェントが以下の 7 カテゴリで質問を開始します。

| カテゴリ | 内容 |
|---------|------|
| scope | 機能のスコープ・境界 |
| functional | 機能要件の詳細 |
| nfr | 非機能要件（性能、セキュリティ等） |
| acceptance | 受入条件 |
| risk | リスクと対策 |
| dependency | 外部依存・前提条件 |
| priority | 優先度・リリース時期 |

応答パターン:
- 推奨回答で OK → **「OK」** と回答
- 修正がある → 修正内容をそのまま回答
- 後回しにしたい → **「スキップ」** と回答
- 番号で一括回答 → **「1. OK 2. XX に変更 3. スキップ」**

### 2.3 進捗の確認

```bash
# セッション進捗
teraflow discovery status discussion-42
```

期待される結果:
```
Discussion: #42
Title: 認証機能の追加
Progress: 12/12 (100%)
Pending: 0
Skipped: 1
Blocked: 0
All branches resolved — ready for requirement confirmation.
```

```bash
# 決定木の構造を確認
teraflow discovery tree discussion-42
```

期待される結果:
```
Discovery tree: discussion-42
- [answered] scope-1: 認証対象のユーザー種別は？
  - [answered] scope-1-1: 外部ユーザーも含むか？
- [answered] functional-1: 認証方式は？
- [skipped] nfr-3: レスポンス時間の上限は？
```

### 2.4 要件確定

全分岐が resolved/skipped になったら、Discussion で「要求確定」とコメントします。

1. CoDD 要件文書が自動生成される（`doc generate --from-draft` 相当）
2. PR が作成される（`review_required: approve` — 人間の承認必須）
3. skipped された質問はリスクセクションに自動追記される
4. PR をレビュー・承認・マージする

> 詳細: [grill-me 設計書](../design/requirements-discovery.md)


## 3. 設計フェーズ

### 3.1 フェーズ遷移

要件定義フェーズ完了後、Issue を作成して設計フェーズに遷移します。

```
Title: Phase transition: basic-design
Labels: phase: basic-design, phase-transition
```

phase-transition ワークフローが自動発火し、以下を実行します:
1. 前フェーズ（要件定義）の完了確認
2. CoDD 整合性チェック
3. `project-state.yml` の更新

### 3.2 Wave 計画の確認

```bash
teraflow plan waves --phase basic-design
```

期待される結果:
```
Phase: basic-design
- Wave 1: acceptance-criteria-adr template=wave-acceptance-criteria.tmpl review=review
- Wave 2: system-design template=wave-system-design.tmpl review=review
  depends_on: [1]
- Wave 3: db-api-design template=wave-db-design.tmpl review=approve
  depends_on: [2]
- Wave 4: ui-design template=wave-ui-design.tmpl review=review
  depends_on: [3]
- Wave 5: implementation-plan template=wave-implementation-plan.tmpl review=approve
  depends_on: [1 2 3 4]
```

### 3.3 設計書の自動生成

```bash
# 全 Wave を実行
teraflow generate --phase basic-design

# 特定の Wave のみ実行
teraflow generate --phase basic-design --wave 1

# プレビュー（ファイル書き込みなし）
teraflow generate --phase basic-design --dry-run

# PR を自動作成
teraflow generate --phase basic-design --create-pr

# カスタムテンプレートを使用
teraflow generate --phase basic-design --template ./my-template.tmpl
```

期待される結果:
```
Phase: basic-design
- Wave 1 acceptance-criteria-adr status=completed
  artifact: docs/design/auth-acceptance.md (design)
  artifact: docs/design/adr-auth.md (design)
- Wave 2 system-design status=completed
  artifact: docs/design/auth-system.md (design)
...
```

各 Wave の成果物は `review_required` の設定に応じてレビューフローに入ります（→ §6 参照）。

### 3.4 フェーズ進捗の確認

```bash
# 現在のフェーズ
teraflow plan phases

# V字モデル全体の状態
teraflow plan status
```

期待される結果（`plan phases`）:
```
Project: my-project
Lifecycle: development
Current phase: basic-design
```

期待される結果（`plan status`）:
```
Project: my-project
Lifecycle: development
Current phase: basic-design
Processes:
- ソフトウェア要件定義: completed
- ソフトウェア方式設計: in_progress
```


## 4. 実装フェーズ

### 4.1 フェーズ遷移

詳細設計完了後、`phase: implementation` ラベルの Issue を作成します。

### 4.2 コード生成

```bash
# 詳細設計書から全モジュールを生成
teraflow implement --design docs/design/auth-api.md

# 特定モジュールのみ生成
teraflow implement --design docs/design/auth-api.md --module internal/auth/handler.go

# 並列数を指定（デフォルト: 3）
teraflow implement --design docs/design/auth-api.md --parallel 5

# プレビュー
teraflow implement --design docs/design/auth-api.md --dry-run

# PR メタデータを生成
teraflow implement --design docs/design/auth-api.md --create-pr
```

期待される結果:
```
Design: docs/design/auth-api.md
Modules: total=3 completed=3 failed=0 skipped=0
- internal/auth/handler.go status=completed review=review
  test: internal/auth/handler_test.go
- internal/auth/middleware.go status=completed review=review
  test: internal/auth/middleware_test.go
- internal/auth/token.go status=completed review=auto
  test: internal/auth/token_test.go
Validation L1-2: valid=true errors=0 warnings=0
```

### 4.3 生成フロー

1. 詳細設計書の `modules` フィールドからモジュール一覧を取得
2. 依存関係のないモジュールを並列生成
3. 各モジュールにテストスケルトンを同時生成
4. CoDD 整合性チェック（Level 1-2）を実行
5. `review_required` に応じて PR を処理


## 5. 検証フェーズ

V字モデルの右側は、左側の設計フェーズと対称関係にあります。

```
要件定義 ←→ 受入テスト    : 要件の受入条件を検証
基本設計 ←→ システムテスト : システム全体の動作を検証
詳細設計 ←→ 結合テスト    : モジュール間の結合を検証
実装     ←→ 単体テスト    : 個々のモジュールを検証
```

### 5.1 単体テスト

実装フェーズで `--module` 指定時にテストスケルトンが同時生成されます（デフォルト有効）。

```bash
# テスト実行
go test ./internal/auth/...
```

### 5.2 結合テスト

詳細設計 Wave 3 で生成されたテスト仕様に基づき、結合テストを実施します。

### 5.3 システムテスト

基本設計で定義されたシステムテスト仕様に基づき、ST を実施します。

### 5.4 受入テスト

要件定義の受入条件（grill-me の acceptance カテゴリ）に基づき、受入判定を行います。受入テストは `review_required: approve` となり、人間の承認が必須です。


## 6. review_required の使い分け

### 6.1 3段階のレビューレベル

| レベル | 動作 | 用途 |
|--------|------|------|
| `auto` | CI 通過後に自動マージ | スケルトンコード、定型テスト、小規模変更 |
| `review` | PR 作成 → 人間レビュー待ち | 設計書、ロジック変更、新機能 |
| `approve` | PR 作成 → 承認者の明示的 approve 必須 | DB 設計、API 設計、セキュリティ関連 |

### 6.2 デフォルト設定

| 成果物 | デフォルトレベル |
|--------|----------------|
| 受入条件 + ADR（Wave 1） | review |
| システム設計書（Wave 2） | review |
| DB設計 + API設計（Wave 3） | approve |
| UI/UX 設計（Wave 4） | review |
| 実装計画（Wave 5） | approve |
| スケルトンコード | auto |
| モジュール実装 | review |
| 統合 PR | approve |
| 要件確定 | approve |

### 6.3 自動昇格

以下の条件に該当する場合、レビューレベルが自動的に昇格します。

| 条件 | 昇格先 |
|------|--------|
| セキュリティモジュール（auth, rbac, crypto） | approve |
| DB スキーマ変更 | approve |
| 外部 API 変更 | review 以上 |
| 影響モジュール数 5 超 | review 以上 |


## 7. 変更管理

### 7.1 整合性チェック

```bash
# 全ノードの完全検証
teraflow validate --full

# 特定フェーズの検証
teraflow validate --phase basic-design

# 個別ノードの検証
teraflow validate --node design:auth-system

# 検証レベルを指定（1-4、デフォルト: 4）
teraflow validate --level 2

# JSON 出力
teraflow validate --full --format json
```

期待される結果:
```
Valid: true
Errors: 0
Warnings: 2
[WARN][L2] design:auth-ui depends_on contains non-existent node: design:auth-ux
[WARN][L3] test:ut-auth-handler verified_by not set
```

検証レベル:
| レベル | 内容 |
|--------|------|
| 1 | Schema — frontmatter の必須フィールド存在確認 |
| 2 | References — depends_on の全ノード存在確認 |
| 3 | Phase — フェーズ整合性（前フェーズ完了確認等） |
| 4 | Graph — GraphRAG による意味的整合性（デフォルト） |

終了コード:
| コード | 意味 |
|--------|------|
| 0 | エラーなし、警告なし |
| 1 | 警告あり、エラーなし |
| 2 | エラーあり |

### 7.2 変更影響分析

上流の CoDD 文書が変更された場合、下流への影響を分析します。

```bash
# 影響分析
teraflow impact req:feature-auth

# --node フラグでも指定可能
teraflow impact --node req:feature-auth

# 探索深度を指定（デフォルト: 2）
teraflow impact --node req:feature-auth --depth 3

# JSON 出力
teraflow impact --node req:feature-auth --output json

# 分析結果からフォローアップ Issue を自動作成
teraflow impact --apply impact-results.json
```

期待される結果:
```
Changed: req:feature-auth
Affected: 5 (gray=1 amber=2 green=2)
- design:auth-system band=amber distance=1 source=depends_on reason=依存チェーン経由
- detail:auth-api band=gray distance=2 source=depends_on reason=高影響判定
- test:it-auth band=amber distance=3 source=depends_on reason=依存チェーン経由
- impl:auth-handler band=green distance=3 source=depends_on reason=影響軽微
- test:ut-auth band=green distance=4 source=depends_on reason=影響軽微
```

### 7.3 バンド（影響度）の意味

| バンド | 意味 | 対応 |
|--------|------|------|
| **Green** | 影響なし、または影響軽微 | 対応不要 |
| **Amber** | 要確認 — 限定的な追従修正が必要な可能性 | レビューして必要に応じて修正 |
| **Gray** | 要再生成 — 整合性破綻リスクが高い | 成果物の再生成が必要 |

### 7.4 `--apply` による自動 Issue 作成

```bash
# 影響分析結果を JSON で保存
teraflow impact --node req:feature-auth --output json > impact-results.json

# 結果に基づいて Issue を自動作成
teraflow impact --apply impact-results.json
```

Amber ノードには「Review impact on ...」、Gray ノードには「Regenerate artifact for ...」の Issue が自動作成されます。


## 8. コマンドリファレンス

v0.5.0 で追加された V字モデルパイプライン関連コマンドの一覧です。

### teraflow generate

Wave ベースで CoDD 成果物を生成する。

```
teraflow generate --phase <phase> [--wave <number>] [--create-pr] [--dry-run] [--template <path>]
```

| フラグ | 型 | デフォルト | 説明 |
|--------|------|-----------|------|
| `--phase` | string | （必須） | 対象フェーズ |
| `--wave` | int | 0 | Wave 番号（0 = 全 Wave） |
| `--create-pr` | bool | false | PR を自動作成（CI ワークフロー連携） |
| `--dry-run` | bool | false | プレビュー（ファイル書き込みなし） |
| `--template` | string | "" | カスタムテンプレートパス |

### teraflow implement

詳細設計書からモジュールを並列コード生成する。

```
teraflow implement --design <path> [--module <path>...] [--parallel <n>] [--create-pr] [--dry-run]
```

| フラグ | 型 | デフォルト | 説明 |
|--------|------|-----------|------|
| `--design` | string | （必須） | 詳細設計書パス |
| `--module` | string[] | 全モジュール | 特定モジュールのみ生成（複数指定可） |
| `--parallel` | int | 3 | 最大並列数 |
| `--create-pr` | bool | false | 統合 PR メタデータを生成 |
| `--dry-run` | bool | false | プレビュー（ファイル書き込みなし） |

### teraflow validate

CoDD 文書の整合性を検証する。

```
teraflow validate [--full] [--phase <phase>] [--node <node-id>] [--level <1-4>] [--format json]
```

| フラグ | 型 | デフォルト | 説明 |
|--------|------|-----------|------|
| `--full` | bool | false | 全ノード検証 |
| `--phase` | string | "" | フェーズ単位の検証 |
| `--node` | string | "" | 個別ノードの検証 |
| `--level` | int | 4 | 最大検証レベル（1-4） |
| `--format` | string | text | 出力形式（text/json） |

> `--phase` と `--node` は同時に指定できません。

### teraflow impact

変更伝播の影響分析を行う。

```
teraflow impact [node-id] [--node <node-id>] [--depth <n>] [--apply <path>] [--output json|text]
```

| フラグ | 型 | デフォルト | 説明 |
|--------|------|-----------|------|
| `--node` | string | "" | 変更されたノード（位置引数でも指定可） |
| `--depth` | int | 2 | 探索深度 |
| `--apply` | string | "" | JSON/NDJSON からフォローアップ Issue を作成 |
| `--output` | string | "" | 出力形式（text/json） |

> `--apply` は `--node` や位置引数と同時に使用できません。

### teraflow plan

V字モデルの実行計画と状態を表示する。

```
teraflow plan waves --phase <phase>
teraflow plan phases
teraflow plan status
```

| サブコマンド | 説明 |
|-------------|------|
| `waves --phase <phase>` | 指定フェーズの Wave 実行計画 |
| `phases` | 現在のフェーズ情報 |
| `status` | V字モデル全体の状態（SLCP-JCF プロセス含む） |

### teraflow discovery

grill-me 要件探索セッションを確認する。

```
teraflow discovery status <discussion-N>
teraflow discovery tree <discussion-N>
```

| サブコマンド | 説明 |
|-------------|------|
| `status <discussion-N>` | セッションの進捗表示（回答数/全体数/完了率） |
| `tree <discussion-N>` | 決定木の構造表示（各分岐のステータス付き） |

> 引数は `discussion-42` または `42` の形式で指定できます。


## 9. トラブルシューティング

### validate でエラーが出る

**症状**: `teraflow validate --full` で `[ERROR][L2] ... depends_on contains non-existent node` が出る。

**原因**: 依存先の CoDD 文書がまだ作成されていない、または `node_id` のタイプミス。

**対処**:
1. `teraflow validate --node <node-id> --level 1` でスキーマレベルを確認
2. 依存先の文書が存在するか確認
3. frontmatter の `node_id` と `depends_on` の値を照合

### generate で "no wave definitions for phase" と出る

**症状**: `teraflow generate --phase xxx` で Wave 定義が見つからない。

**原因**: フェーズ名が正しくない。

**対処**: 有効なフェーズ名を使用してください。
- `requirements`, `basic-design`, `detailed-design`, `implementation`

### implement で provider エラーが出る

**症状**: `create provider: ...` エラー。

**原因**: AI プロバイダーの設定が不足している。

**対処**:
1. `.teraflow.yml` の `providers` セクションを確認
2. 環境変数（`ANTHROPIC_API_KEY` 等）が設定されているか確認
3. `--dry-run` フラグでプロバイダーなしのプレビューが可能

### impact で "gh CLI not found" と出る

**症状**: `teraflow impact --apply` 実行時にエラー。

**原因**: `gh` コマンドがインストールされていない。

**対処**: GitHub CLI をインストールして認証してください。
```bash
# macOS
brew install gh
gh auth login
```

### discovery で "read discovery session" エラーが出る

**症状**: `teraflow discovery status discussion-42` でファイルが見つからない。

**原因**: `.teraflow/discovery/discussion-42.yaml` が存在しない。

**対処**:
1. Discussion に `discovery` ラベルが付いているか確認
2. grill-me エージェントが少なくとも 1 回応答したか確認
3. `.teraflow/discovery/` ディレクトリの中身を確認


## 10. ラベル一覧

V字モデルパイプラインで使用する GitHub ラベルの一覧です。

### フェーズ遷移ラベル

| ラベル | 用途 |
|--------|------|
| `phase-transition` | フェーズ遷移ワークフローのトリガー |
| `phase: requirements` | 要件定義フェーズ |
| `phase: basic-design` | 基本設計フェーズ |
| `phase: detailed-design` | 詳細設計フェーズ |
| `phase: implementation` | 実装フェーズ |
| `phase: unit-test` | 単体テストフェーズ |
| `phase: integration-test` | 結合テストフェーズ |
| `phase: system-test` | システムテストフェーズ |
| `phase: acceptance-test` | 受入テストフェーズ |

### 要件定義ラベル

| ラベル | 用途 |
|--------|------|
| `discovery` | grill-me 要件探索モードのトリガー |
| `要件探索` | discovery の日本語エイリアス |
| `grill` | discovery の短縮エイリアス |
| `requirements` | 従来の壁打ちモード |

### レビューラベル

| ラベル | 用途 |
|--------|------|
| `needs-approval` | `review_required: approve` の PR に自動付与 |

### 変更管理ラベル

| ラベル | 用途 |
|--------|------|
| `impact` | 変更影響分析の Issue |
| `amber` | Amber バンド（要レビュー）の Issue |
| `gray` | Gray バンド（要再生成）の Issue |
| `regeneration` | 成果物再生成が必要な Issue |

### 実装ラベル

| ラベル | 用途 |
|--------|------|
| `agent-implement` | AI 実装エージェントのトリガー |
