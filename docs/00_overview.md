---
codd:
  node_id: "req:teraflow-overview"
  title: "teraflow要件定義概要 — 目的・3層責務モデル・ライフサイクルステージ"
  depends_on: []
---

# teraflow — Terasoluna準拠 AI駆動開発パイプライン CLI

## 0. 本ドキュメントの目的

本ドキュメントは、Terasoluna フレームワークを主体としたシステム開発の **ライフサイクル全体** を GitHub 単一リポジトリ上で管理し、AIエージェントによる自律開発パイプラインを段階的に導入するための要件定義書である。

初期開発だけでなく、リリース後の運用・継続的改善・保守・廃止に至るまで、サービスのライフサイクル全体を一貫して管理できるCLIツール `teraflow` の実装を目的とする。

### 参照手法・ツール

| 手法/ツール | 役割 | 参照 |
|---|---|---|
| Terasoluna | 3層アーキテクチャ（app / domain / infra）によるレイヤー分離。言語非依存の抽象的開発ガイドライン | NTTデータ Terasoluna Framework |
| CoDD | 成果物間の依存グラフ構築、変更影響分析（`codd scan` / `codd impact`） | https://github.com/yohey-w/codd-dev |
| ハーネスエンジニアリング | AIエージェントの品質担保。Context供給 / アーキテクチャ制約 / フィードバックループの3層 | OpenAI / Martin Fowler |

### 重要な設計原則

- **Terasoluna は言語非依存**: 特定のプログラミング言語・フレームワークに依存しない。3層アーキテクチャの概念とガイドラインとして扱う。CLI も特定言語を前提としない。
- **ライフサイクル全体の管理**: 初期開発だけでなく、リリース後の運用・改善・保守・廃止まで継続的にアプローチする。
- **段階的導入**: 全機能を一度に有効化するのではなく、ステージ・フェーズごとに必要な機能だけを有効化する。
- **GitHub完結**: 全ての管理・自動化をGitHub（Issues, Discussions, Actions, Projects, Pages）上で完結させる。
- **3層責務モデル（Discussion / Issue / Files）**: Discussion は非構造的な入口、Issue は一時的な構造化作業場（確定後クローズ）、リポジトリのファイル（docs/）が唯一の正の記録（Source of Truth）。トレーサビリティは Issue 間のリンクではなく、ファイル間の CoDD frontmatter で管理する。

---

## 1. 3層責務モデル（Discussion / Issue / Files）

巨大プロジェクトでは Issue が数百〜数千に膨らみ管理不能になる。これを防ぐため、GitHub の3つの機能に明確な責務を割り当てる。

### 1.1 各層の責務

```
Discussion（非構造的な入口）
  ├── 非エンジニアの相談・要望投稿
  ├── 気軽なブレスト・雑談
  └── 要求フェーズでのAI対話
      → 「要求確定」で Issue 化

Issue（構造化された一時作業場）
  ├── フェーズに応じたテンプレートで起票
  ├── AI対話で内容を詰める
  ├── 人間同士の議論・レビュー
  └── 「確定」→ PR作成 → マージ → Issue自動クローズ
      ※ Open Issue は「現在作業中」のもののみ
      ※ 確定後は速やかにクローズ

Files / docs/（正の記録 = Source of Truth）
  ├── 確定済みの全成果物（要求/要件/設計/テスト計画等）
  ├── index.md で一覧管理
  ├── CoDD frontmatter でファイル間のトレーサビリティ
  └── 全ての意思決定はファイルに残る
```

### 1.2 全フェーズ共通の確定フロー

どのフェーズでも同じパターンで成果物を確定する。

```
1. Issue 作成（フェーズに応じたテンプレートで起票）
       │
2. Issue 上で AI 対話 or 人間同士の議論で内容を詰める
       │
3. 内容が固まったらユーザが「確定」とコメント
       │
4. Actions が自動起動
   ├── AI が会話履歴から成果物ファイルを生成
   │     例: REQ-0010.md / REQDEF-0020.md / BD-0030.md / DD-0040.md
   ├── CoDD frontmatter を付与（depends_on で上流成果物にリンク）
   ├── docs/ 配下にファイルを配置
   ├── index.md を自動更新
   └── PR を作成
       │
5. PR レビュー → マージ
       │
6. Issue を自動クローズ
   └── クローズコメントに成果物ファイルへのリンクを記載
```

### 1.3 フェーズごとの成果物ファイル

| フェーズ | ファイル命名 | 配置先 | 例 |
|---|---|---|---|
| 要求整理 | `REQ-{NNNN}.md` | `docs/01_requirements/` | `REQ-0010.md` |
| 要件定義 | `REQDEF-{NNNN}.md` | `docs/02_requirement-definition/` | `REQDEF-0020.md` |
| 基本設計 | `BD-{NNNN}.md` | `docs/03_basic-design/` | `BD-0030.md` |
| 詳細設計 | `DD-{NNNN}.md` | `docs/04_detailed-design/` | `DD-0040.md` |
| 実装 | ソースコード + PR | `src/` | — |
| テスト | `TEST-{NNNN}.md` | `docs/05_test/` | `TEST-0050.md` |

各ディレクトリに `index.md` を配置し、確定済み成果物の一覧を自動生成する。

### 1.4 トレーサビリティ（ファイルベース）

Issue 間のリンクではなく、**ファイル間の CoDD frontmatter** でトレーサビリティを管理する。

```yaml
# docs/04_detailed-design/DD-0040.md
---
codd:
  node_id: "detail:DD-0040"
  depends_on:
    - id: "design:BD-0030"
      relation: derives_from
    - id: "reqdef:REQDEF-0020"
      relation: implements
    - id: "req:REQ-0010"
      relation: traces_to
source_issue: 40              # 元のIssue番号（参考情報として記録）
confirmed_at: "2026-06-15"
confirmed_by: "@engineer-a"
---
```

**上流→下流の追跡**:

```
REQ-0010.md（要求）
 └── REQDEF-0020.md（要件）    ← depends_on: REQ-0010
      └── BD-0030.md（基本設計）← depends_on: REQDEF-0020
           └── DD-0040.md（詳細設計）← depends_on: BD-0030
                └── src/ のコード ← depends_on: DD-0040
```

Issue はクローズされても、ファイルに `source_issue` が記録されているため必要なら遡れる。ただし日常的な追跡はファイル間の `depends_on` で完結する。

### 1.5 進捗管理の方式

Issue の Open/Close 数ではなく、**成果物ファイルの確定数**で進捗を測る。

```
フェーズ進捗 = 確定済みファイル数 / 想定ファイル数

例:
  要件定義フェーズ:
    docs/02_requirement-definition/ に REQDEF-*.md が 18件
    想定: 25件（master-schedule.yml の planned_artifacts）
    進捗: 18/25 = 72%
```

ゲート条件も同様に、Issue のクローズ数ではなくファイルの存在で判定する。

```yaml
# ゲート条件の例
phases:
  要求整理:
    gate_conditions:
      - type: files_count_match
        path: "docs/01_requirements/REQ-*.md"
        description: "全要求が確定済みファイルとして存在する"
      - type: file_exists
        path: "docs/01_requirements/index.md"
      - type: gate_issue_closed
        issue_label: "gate:要求整理"
```

### 1.6 Issue の最小化ルール

```
常時 Open であるべき Issue:
  ・現在作業中の確定前 Issue（フェーズ内で AI 対話中のもの）
  ・ゲート Issue（フェーズ完了判定用、各フェーズに1つ）
  ・手戻り Issue（未解決のもの）
  ・障害 Issue（未解決のもの）

クローズされるべき Issue:
  ・確定済み → 成果物ファイルが PR でマージされた時点で自動クローズ
  ・却下 → not planned でクローズ
  ・フェーズガードで拒否 → 自動クローズ

結果:
  巨大プロジェクトでも Open Issue は常に数十件程度に収まる。
  全ての確定済み情報は docs/ 配下のファイルに永続化されている。
```

---

## 2. ライフサイクルステージ

teraflow はプロジェクトのライフサイクルを **6つのステージ** で管理する。各ステージは内部に固有の **フェーズ** を持ち、フェーズの構成・スキップ可否・管理方式がステージごとに異なる。

### 1.1 ステージ一覧

```
① 初期開発          最初のリリースに向けた開発
     │
     ▼
② 移行・リリース     本番環境への移行、稼働開始
     │
     ▼
③ 運用              稼働中の監視・障害対応
     │
     ▼  ┐
④ 継続的改善         機能追加・改修（ミニ開発サイクルの繰り返し）
     │  │ 繰り返し
     ▼  ┘
⑤ 保守              技術的負債返済・セキュリティ・EOL対応
     │
     ▼
⑥ 廃止              サービス終了・データ移行・後継への引継ぎ
```

### 1.2 各ステージの定義

#### ① 初期開発

最初のリリースに向けた、最も重い開発サイクル。全フェーズを順に通る。

| 項目 | 内容 |
|---|---|
| 内部フェーズ | 要求整理 → 要件定義 → 基本設計 → 詳細設計 → 実装 → テスト |
| フェーズスキップ | 不可 |
| 主体 | 非エンジニア（上流）+ エンジニア（下流） |
| 終了条件 | 全フェーズ完了 + リリース判定合格 |

#### ② 移行・リリース

本番環境への移行と稼働開始。ここでの失敗はサービスに直結するため独立ステージとする。

| 項目 | 内容 |
|---|---|
| 内部フェーズ | 移行計画 → 環境構築 → データ移行 → リハーサル → 本番切替 → 安定化確認 |
| フェーズスキップ | 不可 |
| 主体 | エンジニア + インフラ担当 |
| 終了条件 | 本番稼働確認 + 安定化期間の無障害確認 |
| 必須成果物 | 移行計画書、切り戻し計画書、リリース判定チェックリスト |

#### ③ 運用

稼働中のシステムを安定的に動かし続ける。フェーズ管理ではなくイベント駆動で管理する。④継続的改善と並行して常時稼働する。

| 項目 | 内容 |
|---|---|
| 管理方式 | イベント駆動（フェーズ順序なし） |
| Issue種別 | 障害報告 / 問い合わせ / 定期作業 / 監視アラート |
| 主体 | 運用チーム / エンジニア |
| 特徴 | 終了条件なし（サービス稼働中は常時継続） |

#### ④ 継続的改善

リリース後の機能追加・改修。初期開発と同じフェーズ構成だが、スコープが小さくサイクルが短い。Milestone単位でサイクルを区切り、複数サイクルが並行して走ることがある。③運用と並行して実行される。

| 項目 | 内容 |
|---|---|
| 内部フェーズ | 要求整理 → 要件定義 → 基本設計 → 詳細設計 → 実装 → テスト → リリース |
| フェーズスキップ | 可（小規模改修では要求→実装→テスト→リリース等） |
| 主体 | 非エンジニア（要求）+ エンジニア（実装） |
| サイクル管理 | Milestone単位（例: "v1.1 注文検索機能追加"） |
| 特徴 | 稼働中システムへの変更のため、影響分析（CoDD）とリグレッションテストがより重要 |

#### ⑤ 保守

機能追加を伴わない維持管理。④継続的改善とは予算・体制・優先度の判断基準が異なるため分離する。

| 項目 | 内容 |
|---|---|
| 内部フェーズ | 調査 → 計画 → 実施 → 検証 |
| フェーズスキップ | 可 |
| Issue種別 | セキュリティパッチ / ライブラリ更新 / リファクタリング / EOL対応 |
| 主体 | エンジニア |
| 特徴 | 定期スコアリングの結果に基づき Issue を自動起票することがある |

#### ⑥ 廃止

サービス終了に伴う計画的な停止と後処理。多くのプロジェクトが計画しないが、最初から定義しておくことでデータ設計やAPI設計の段階で移行可能性を考慮できる。

| 項目 | 内容 |
|---|---|
| 内部フェーズ | 廃止計画 → 告知・移行支援 → サービス停止 → アーカイブ |
| フェーズスキップ | 不可 |
| 主体 | PM + エンジニア |
| 必須成果物 | 廃止計画書、データエクスポート手順書、後継システム引継ぎ資料 |

### 1.3 ステージ間の遷移ルール

ステージ遷移は `teraflow stage advance` で実行する。各ステージにはゲート条件を定義する。

**特殊な遷移**:
- ③運用 と ④継続的改善 は **並行して稼働** する。③に遷移した時点で④も自動的に有効化される。
- ⑤保守 は ③④と **並行して稼働** する。明示的に有効化する。
- ⑥廃止 への遷移は ③④⑤ を停止する。

```
ステージ状態の例:

  リリース直後:
    ③ 運用: active
    ④ 継続的改善: active
    ⑤ 保守: inactive（必要に応じて有効化）

  安定運用期:
    ③ 運用: active
    ④ 継続的改善: active
    ⑤ 保守: active

  サービス終了決定後:
    ③ 運用: active（終了まで継続）
    ④ 継続的改善: frozen（新規サイクル停止）
    ⑤ 保守: frozen（最低限のみ）
    ⑥ 廃止: active
```

### 1.4 ステージ状態ファイル

```yaml
# .github/project-state.yml

lifecycle:
  current_stages:             # 複数ステージが同時にactiveになりうる
    - stage: "運用"
      status: "active"
      started_at: "2026-08-16"
    - stage: "継続的改善"
      status: "active"
      started_at: "2026-08-16"
    - stage: "保守"
      status: "inactive"

  stage_history:
    - stage: "初期開発"
      started_at: "2026-04-01"
      ended_at: "2026-08-01"
    - stage: "移行・リリース"
      started_at: "2026-08-01"
      ended_at: "2026-08-15"

  # 現在のフェーズ（アクティブなステージごとに管理）
  active_phases:
    初期開発:
      current_phase: null       # 完了済み
    継続的改善:
      current_cycle: "v1.1"     # Milestone名
      current_phase: "実装"
    保守:
      current_phase: null       # inactive

  # フェーズ履歴（ステージ単位で記録）
  phase_history:
    初期開発:
      - phase: "要求整理"
        started_at: "2026-04-01"
        ended_at: "2026-04-18"
        result: "completed"          # completed / reverted / skipped

      - phase: "要件定義"
        started_at: "2026-04-21"
        ended_at: "2026-05-09"
        result: "completed"

      - phase: "基本設計"
        started_at: "2026-05-12"
        ended_at: "2026-05-20"
        result: "reverted"           # 手戻りで中断
        reverted_to: "要件定義"
        rework_issue: "#42"
        rework_reason: "注文検索の条件が未定義"

      - phase: "要件定義"              # 2回目の要件定義
        started_at: "2026-05-20"
        ended_at: "2026-05-25"
        result: "completed"
        triggered_by: "rework"       # 手戻りで開始
        rework_issue: "#42"

      - phase: "基本設計"              # 2回目の基本設計
        started_at: "2026-05-26"
        ended_at: null               # 現在進行中
        result: null
        triggered_by: "rework"
        rework_issue: "#42"
```

---

