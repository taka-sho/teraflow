# 共通フレーム（SLCP-JCF2013）移行影響分析

## 1. 共通フレーム（SLCP-JCF2013）プロセス体系概要

### 1.1 概要

共通フレーム2013は、IPA（情報処理推進機構）が策定したソフトウェアライフサイクルプロセスの日本版ガイドライン。国際規格 ISO/IEC 12207:2008（JIS X 0160:2012）に準拠し、ソフトウェア開発における発注者・受注者間の共通認識を形成する。

| 項目 | 内容 |
|------|------|
| 正式名称 | Software Life Cycle Process - Japan Common Frame 2013 |
| 国際規格 | ISO/IEC 12207:2008 準拠 |
| 構造 | プロセス → アクティビティ → タスク の3階層 |
| 適用範囲 | ソフトウェア + ハードウェアを含むシステム全体 |

### 1.2 プロセス体系（8区分）

| 区分 | カテゴリ | 主要プロセス |
|------|---------|------------|
| 第1区分 | **合意プロセス** | 1.1 取得、1.2 供給、1.3 合意・契約変更管理 |
| 第2区分 | **テクニカルプロセス** | 2.1 企画、2.2 要件定義、2.3 システム開発、2.4 ソフトウェア実装、2.5 ハードウェア実装、2.6 保守 |
| 第3区分 | **運用・サービスプロセス** | 3.1 運用、3.2 廃棄、3.3 サービスマネジメント |
| 第4区分 | **支援プロセス** | 4.1 文書化、4.2 品質保証、4.3 検証、4.4 妥当性確認、4.5 共同レビュー、4.6 監査、4.7 問題解決 |
| 第5区分 | **プロジェクトプロセス** | 5.1 計画、5.2 アセスメント・制御、5.3 意思決定管理、5.4 リスク管理、5.5 構成管理、5.6 ソフトウェア構成管理、5.7 情報管理、5.8 測定 |
| 第6区分 | **組織プロジェクトイネーブリング** | ライフサイクルモデル管理、インフラ管理、改善、人的資源管理 |
| 第7区分 | **プロセスビュー** | ユーザビリティ関連 |
| 第8区分 | **テーラリングプロセス** | 状況に応じた修整 |

### 1.3 テクニカルプロセス詳細（teraflowに最も関連）

#### 2.1 企画プロセス

| アクティビティ | 内容 |
|--------------|------|
| 2.1.1 システム化構想の立案 | 経営ニーズ分析、事業環境調査、現行システム調査、IT動向調査 |
| 2.1.2 システム化計画の立案 | 全体方針策定、スケジュール、費用対効果 |

#### 2.2 要件定義プロセス

| アクティビティ | 内容 |
|--------------|------|
| 2.2.1 プロセス開始の準備 | 要件定義の範囲と方法の決定 |
| 2.2.2 利害関係者の識別 | ステークホルダー特定 |
| 2.2.3 要件の識別 | 要件抽出、制約条件定義 |
| 2.2.4 要件の評価 | 実現可能性、一貫性、追跡可能性の評価 |
| 2.2.5 要件の合意 | ステークホルダー間の合意形成 |
| 2.2.6 要件の記録 | 要件仕様書の作成 |

#### 2.3 システム開発プロセス

| アクティビティ | 内容 |
|--------------|------|
| 2.3.1 開始準備 | 開発戦略、計画策定 |
| 2.3.2 システム要件定義 | システム要件の分析・仕様化 |
| 2.3.3 システム方式設計 | アーキテクチャ設計、HW/SW/人の分担 |
| 2.3.4 実装 | 構築・コーディング |
| 2.3.5 システム結合 | コンポーネント結合テスト |
| 2.3.6 システム適格性確認テスト | 要件充足の検証 |
| 2.3.7 システム導入 | 本番環境への導入 |
| 2.3.8 システム受入れ支援 | 受入テスト支援 |

#### 2.4 ソフトウェア実装プロセス

| アクティビティ | 内容 |
|--------------|------|
| 2.4.1 開始準備 | ソフトウェア開発計画 |
| 2.4.2 ソフトウェア要件定義 | SW要件の分析・仕様化 |
| 2.4.3 ソフトウェア方式設計 | SW構造・コンポーネント設計、外部IF設計、DB最上位設計 |
| 2.4.4 ソフトウェア詳細設計 | プログラム詳細設計、DB詳細設計 |
| 2.4.5 ソフトウェア構築 | コーディング + 単体テスト |
| 2.4.6 ソフトウェア結合 | 結合テスト |
| 2.4.7 ソフトウェア適格性確認テスト | 要件充足の確認テスト |
| 2.4.8 ソフトウェア導入 | 導入・移行 |
| 2.4.9 ソフトウェア受入れ支援 | 受入テスト支援 |

#### 2.6 保守プロセス

| アクティビティ | 内容 |
|--------------|------|
| 2.6.1 プロセスの開始 | 保守計画策定 |
| 2.6.2 問題把握及び修正の分析 | 問題分析、修正範囲の特定 |
| 2.6.3 修正の実施 | 修正開発・テスト |
| 2.6.4 保守レビュー | レビュー・受入れ |
| 2.6.5 運用テスト及び移行の支援 | 修正版の移行支援 |

---

## 2. teraflow現行概念 vs 共通フレーム 対応表

### 2.1 ステージ対応

| teraflow現在 | 共通フレーム対応候補 | 変更要否 | 備考 |
|-------------|-------------------|---------|------|
| **ステージ全体** (6ステージ) | テクニカルプロセス(第2区分) + 運用・サービスプロセス(第3区分) | **要** | 概念名は変更不要だが、プロセス番号との対応表が必要 |
| ① 初期開発 | 2.3 システム開発 + 2.4 ソフトウェア実装 | 不要 | 名称「初期開発」は共通フレームにない用語だが、2.3+2.4の複合に相当。変更不要 |
| ② 移行・リリース | 2.3.7 システム導入 + 2.3.8 受入れ支援 | 不要 | 共通フレームでは「導入」プロセス。teraflowの「移行・リリース」の方が直感的 |
| ③ 運用 | 3.1 運用プロセス | 不要 | 完全一致 |
| ④ 継続的改善 | 2.6 保守プロセス（是正保守/適応保守/完全化保守） | **要検討** | 共通フレームでは「保守」に包含される。teraflowの「継続的改善」は保守の一部 |
| ⑤ 保守 | 2.6 保守プロセス（予防保守） | **要検討** | teraflowの「保守」と④「継続的改善」は共通フレームでは同一プロセス |
| ⑥ 廃止 | 3.2 廃棄プロセス | 不要 | ほぼ一致。「廃止」→「廃棄」の用語差のみ |

### 2.2 フェーズ対応

| teraflow現在（初期開発のフェーズ） | 共通フレーム対応 | 変更要否 | 備考 |
|--------------------------------|----------------|---------|------|
| 要求整理 | 2.1 企画プロセス | **要** | 共通フレームでは「企画」。teraflowの「要求整理」は企画の一部 |
| 要件定義 | 2.2 要件定義プロセス | 不要 | 完全一致 |
| 基本設計 | 2.4.3 ソフトウェア方式設計 | **要検討** | 共通フレームでは「方式設計」。日本のSI業界では「基本設計」が通称 |
| 詳細設計 | 2.4.4 ソフトウェア詳細設計 | 不要 | 完全一致 |
| 実装 | 2.4.5 ソフトウェア構築 | **要検討** | 共通フレームでは「構築」（構築=コーディング+単体テスト） |
| テスト | 2.4.6 結合 + 2.4.7 適格性確認テスト | **要** | 共通フレームではテストが2段階に分かれている |

### 2.3 機能概念対応

| teraflow現在 | 共通フレーム対応 | 変更要否 | 備考 |
|-------------|----------------|---------|------|
| gate（ゲート条件） | 4.5 共同レビュー / 4.4 妥当性確認 | 不要 | ゲート概念は共通フレームの「レビュー」「妥当性確認」に相当。用語変更不要 |
| rework（手戻り） | 該当なし（共通フレームに手戻り管理の明示的プロセスなし） | 不要 | teraflow独自の価値。共通フレームにない＝変更不要 |
| incident（障害） | 4.7 問題解決プロセス | 不要 | 「障害」→「問題解決」の対応。teraflowのincidentは問題解決の一種 |
| harness score（保守スコア） | 5.8 測定プロセス | 不要 | teraflow独自メトリクス。共通フレームの「測定」に包含可能 |
| changelog | 4.1 文書化プロセス | 不要 | 変更記録は文書化の一部 |
| trace（トレーサビリティ） | 5.5 構成管理 + 4.3 検証 | 不要 | CoDD依存グラフ管理は構成管理+検証に該当 |
| group（グループ管理） | 該当なし | 不要 | teraflow独自概念 |
| cycle（サイクル管理） | 該当なし | 不要 | 反復開発はテーラリング（第8区分）で対応 |

### 2.4 CLIコマンド対応

| コマンドカテゴリ | 共通フレーム対応 | 変更要否 | 備考 |
|----------------|----------------|---------|------|
| `teraflow stage *` | ライフサイクルプロセス遷移 | **要検討** | stage → process への改名案あり |
| `teraflow phase *` | プロセス内アクティビティ遷移 | **要検討** | phase → activity への改名案あり |
| `teraflow init` | — | 不要 | ツール固有 |
| `teraflow setup *` | — | 不要 | ツール固有 |
| `teraflow group *` | — | 不要 | teraflow独自 |
| `teraflow rework *` | — | 不要 | teraflow独自価値 |
| `teraflow incident *` | 4.7 問題解決 | 不要 | incident は一般的用語 |
| `teraflow schedule *` | 5.1 プロジェクト計画 | 不要 | スケジュール管理 |
| `teraflow harness *` | 5.8 測定 | 不要 | teraflow独自 |
| `teraflow trace *` | 5.5 構成管理 | 不要 | CoDD固有 |
| `teraflow agent *` | — | 不要 | AI Agent固有 |
| `teraflow dashboard *` | — | 不要 | ツール固有 |
| `teraflow doctor` | — | 不要 | ツール固有 |

---

## 3. 変更影響分析

### 3.1 変更が必要なファイル一覧

| ファイル | 変更内容 | 規模 |
|---------|---------|------|
| **概念・用語変更（必須）** | | |
| `docs/00_overview.md` | 「Terasoluna準拠」→「共通フレーム準拠」、参照手法テーブル更新 | M |
| `docs/00_overview.md` (ステージ定義) | 6ステージの説明に共通フレームプロセス番号を併記 | M |
| `docs/00_overview.md` (フェーズ定義) | フェーズ名に共通フレームアクティビティ番号を併記 | M |
| `docs/01_cli-and-project-management.md` | 概念説明の用語更新 | S |
| `docs/index.md` | タイトル・説明文更新 | S |
| **ADR更新** | | |
| `docs/adr/ADR-001-implementation-language.md` | コンテキスト中のTerasoluna言及を更新 | S |
| `docs/adr/ADR-004-data-github-integration.md` | ステージ/フェーズ定義の参照元を共通フレームに更新 | S |
| **設計書更新** | | |
| `docs/design/system-overview.md` | コンポーネント説明の概念参照を更新 | S |
| `docs/design/cli-interface.md` | コマンド説明の概念参照を更新 | S |
| `docs/design/phase2-system-design.md` | フェーズ/ステージ参照を更新 | S |
| `docs/design/phase2-implementation-plan.md` | スコープ説明更新 | S |
| **ソースコード（案Cの場合のみ）** | | |
| `cmd/stage.go` | エイリアスコマンド `teraflow process` 追加 | S |
| `cmd/phase.go` | エイリアスコマンド `teraflow activity` 追加 | S |
| **設定ファイル** | | |
| `.github/teraflow.yml` | 概念参照コメント更新（機能変更なし） | S |
| **テスト** | | |
| 既存テストファイル群 | テスト内の文字列リテラル更新（概念名に依存するもの） | S〜M |
| **新規作成** | | |
| `docs/guide/slcp-jcf-mapping.md` (新規) | 共通フレーム対応表（ユーザー向け参照資料） | M |

### 3.2 影響範囲サマリ

| カテゴリ | ファイル数 | 平均規模 | 合計工数 |
|---------|----------|---------|---------|
| ドキュメント（概念更新） | 11 | S | S〜M × 11 |
| ソースコード（案C限定） | 2 | S | S × 2 |
| テスト | 3〜5 | S | S × 5 |
| 新規ドキュメント | 1 | M | M × 1 |
| **合計** | **17〜19** | | |

---

## 4. 移行方針の提案（3案）

### 案A: 完全移行（Terasoluna概念を廃止し共通フレームに統一）

**内容**: docs内のすべてのTerasoluna参照を共通フレームに置換。ステージ名・フェーズ名を共通フレーム用語に変更。CLIコマンド名も変更（`stage` → `process`、`phase` → `activity`）。

| 項目 | 評価 |
|------|------|
| **メリット** | 概念の一貫性が最も高い。共通フレーム準拠を明確に打ち出せる。日本のSI業界での親和性向上 |
| **デメリット** | 破壊的変更が大きい。Phase1既存ユーザーのCLIコマンドが変わる（`stage advance` → `process advance`）。Phase2の16ワークフローも全面改修。テスト全面書き直し |
| **推定工数** | L（大規模）。全ドキュメント+全ソースコード+全テスト+Phase2ワークフローの改修 |
| **リスク** | Phase2開発中の大規模リネーム。バグ混入リスク高 |

### 案B: 部分移行（共通フレーム概念を追加しつつ既存を維持）

**内容**: docs内の概念説明を「共通フレーム準拠」に更新。ステージ/フェーズの定義に共通フレームプロセス番号を併記。CLIコマンド名は変更しない。新規ドキュメント `docs/guide/slcp-jcf-mapping.md` で対応表を提供。

| 項目 | 評価 |
|------|------|
| **メリット** | 破壊的変更なし。既存ユーザーへの影響ゼロ。共通フレームとの対応関係が明確。ドキュメントのみの更新で完了 |
| **デメリット** | teraflow独自用語（stage/phase）と共通フレーム用語（プロセス/アクティビティ）が併存し、初見で混乱する可能性 |
| **推定工数** | M（中規模）。ドキュメント11ファイル更新 + 新規1ファイル |
| **リスク** | 低。ソースコード変更なし |

### 案C: 別名エイリアス（共通フレーム用語を追加し既存コマンドもエイリアスとして維持）

**内容**: 案Bのドキュメント更新に加え、CLIコマンドにエイリアスを追加。`teraflow process advance` = `teraflow stage advance`（両方動作）。`.github/teraflow.yml` で `terminology: slcp-jcf` 設定時に出力メッセージを共通フレーム用語に切り替え。

| 項目 | 評価 |
|------|------|
| **メリット** | 既存ユーザー影響ゼロ + 共通フレームユーザーに自然なCLI体験。将来的に完全移行も可能（エイリアスを段階的に推奨→旧名を非推奨化） |
| **デメリット** | コマンド体系が2倍になりメンテナンスコスト増。ドキュメントでどちらの用語を主にするか判断が必要 |
| **推定工数** | M〜L。ドキュメント更新 + cmd/stage.go, cmd/phase.go エイリアス追加 + テスト追加 + terminology設定実装 |
| **リスク** | 中。エイリアス間の挙動不整合リスク。ただしcobra のエイリアス機能で実装は容易 |

---

## 5. 推奨案と根拠

### 推奨: 案B（部分移行）

#### 根拠

1. **Phase2開発への影響最小化**: Phase2の16ワークフロー（TF-001〜016）が `stage` / `phase` の概念に依存している。cmd_101で策定した34タスクの実装計画を変更する必要がない

2. **破壊的変更の回避**: teraflow v0.4.0+ の既存ユーザー（CLI操作）に影響なし。`teraflow stage advance` は引き続き動作。テストカバレッジ85%を維持（テスト改修不要）

3. **共通フレーム準拠の実質的達成**: 概念の対応関係が明確であれば、コマンド名が異なっていても「共通フレーム準拠」を謳える。ISO/IEC 12207はプロセス名を強制しない（テーラリングが前提）

4. **将来の拡張パス**: 案Bで対応表を確立した後、将来的に案Cへの移行が容易。逆に案Aで全面改修すると後戻りが困難

5. **工数対効果**: 案Bは11ファイル更新 + 1ファイル新規のM規模。案Aの全面改修（L規模）に比べて1/3以下の工数で同等の対外的効果

#### teraflow独自概念の価値保持

共通フレームに**存在しない**teraflow独自概念は、差別化ポイントとして維持すべき:

- **手戻り管理（rework）**: 共通フレームにない。競合調査（cmd_095）でも類似機能ゼロ
- **保守スコア（harness score）**: 測定プロセスの発展形。独自KPI
- **グループ管理（group）**: 大規模開発向け機能分割。共通フレームの範囲外
- **CoDD依存グラフ（trace）**: 構成管理+検証の発展形。独自実装

これらを共通フレーム用語に無理に置き換えると差別化を失う。

---

## 6. Step 2以降の実行計画（案）

### 前提: 案B（部分移行）承認後

| Step | 内容 | 担当 | 規模 | 依存 |
|------|------|------|------|------|
| Step 1 | 殿の承認ゲート（本文書レビュー） | 殿 | — | Step 0完了 |
| Step 2 | `docs/guide/slcp-jcf-mapping.md` 新規作成（共通フレーム対応表） | ashigaru | M | Step 1承認 |
| Step 3 | `docs/00_overview.md` 更新（概念説明の用語更新、プロセス番号併記） | ashigaru | M | Step 2 |
| Step 4 | ADR群・設計書群のTerasoluna参照を共通フレーム参照に更新（11ファイル） | ashigaru | M | Step 3 |
| Step 5 | `golden-principles.md` / `getting-started.md` 等ガイド類の更新 | ashigaru | S | Step 4 |
| Step 6 | codd scan 実施 + 依存グラフ整合性確認 | gunshi | S | Step 5 |
| Step 7 | 全テストPASS確認（ソースコード変更なしのため自動PASS想定） | ashigaru | S | Step 5 |

**見積合計**: M×3 + S×3（足軽2〜3セッション、軍師1セッション）

### 注意事項

- Step 2〜5はPhase2 Wave実装と並列実行可能（ソースコード変更なし）
- 「Terasoluna準拠」→「共通フレーム（SLCP-JCF2013）準拠」の用語変更が主
- `docs/00_overview.md` のタイトル「teraflow — Terasoluna準拠 AI駆動開発パイプライン CLI」は「teraflow — 共通フレーム準拠 AI駆動開発パイプライン CLI」に変更

---

## 7. プロセス可視化・ロール別次アクション要件（Step 0b追加分析）

### 背景

殿からの追加要件:
> 「ツール側がプロセスの状態を明確にし、利用者（PJチーム全員）が現在の状況を把握し次に何をすべきかが分かりやすいものにすること」

現行の `project-state.yml` は `current_stage`（文字列1つ）と `phases.current`（文字列1つ）のみ。ダッシュボードも現在位置の表示に留まり、**完了済みプロセスの追跡**、**ロール別ガイダンス**、**全体進捗の可視化**が欠けている。

### A. プロセス状態自動追跡の設計案

#### A-1. project-state.yml 拡張案

現行:
```yaml
lifecycle:
  current_stage: initial_development
phases:
  current: detailed_design
```

拡張案:
```yaml
lifecycle:
  current_stage: initial_development

processes:
  planning:              # 2.1 企画
    status: completed    # completed | in_progress | not_started | skipped
    started_at: "2026-01-15"
    completed_at: "2026-01-20"
    gate_passed: true
  requirements:          # 2.2 要件定義
    status: completed
    started_at: "2026-01-21"
    completed_at: "2026-02-05"
    gate_passed: true
  basic_design:          # 2.4.3 ソフトウェア方式設計
    status: completed
    started_at: "2026-02-06"
    completed_at: "2026-02-20"
    gate_passed: true
  detailed_design:       # 2.4.4 ソフトウェア詳細設計
    status: in_progress
    started_at: "2026-02-21"
    completed_at: null
    gate_passed: false
    completion_estimate: "2026-03-10"
  implementation:        # 2.4.5 ソフトウェア構築
    status: not_started
  testing:               # 2.4.6-7 結合テスト+適格性確認
    status: not_started
  release:               # 2.3.7-8 導入+受入れ
    status: not_started

# ゲート条件の自動検出ルール
gate_rules:
  requirements:
    conditions:
      - type: issue_label
        label: "requirements"
        state: closed          # 全requirements Issueがclosedか
      - type: document_exists
        path: "docs/requirements.md"
    auto_detect: true          # GitHub Actions経由で自動評価
  basic_design:
    conditions:
      - type: pr_merged
        label: "design-review"
      - type: document_exists
        path: "docs/design/"
    auto_detect: true
  detailed_design:
    conditions:
      - type: issue_ratio
        label: "detailed-design"
        threshold: 0.9        # 90%以上のIssueがcloseされたら完了候補
      - type: manual_approval  # PM手動承認も必要
    auto_detect: partial       # 自動検出+手動承認のハイブリッド
  testing:
    conditions:
      - type: test_pass_rate
        threshold: 1.0         # テスト全件PASS
      - type: coverage
        threshold: 0.80        # カバレッジ80%以上
    auto_detect: true
```

#### A-2. 既存stage/phase概念との統合方法

案Bの「部分移行」前提で、既存のstage/phaseを壊さずにprocesses層を追加する:

```
[既存] stage (6段階: initial_development → retirement)
  └── [既存] phase (6段階: requirements → integration_test)
        └── [新規] process status (各プロセスの完了/進行中/未着手)
              └── [新規] gate_rules (自動検出条件)
```

- `stage` と `phase` のコマンドはそのまま動作（後方互換）
- `processes` セクションは `teraflow process` 新コマンドで管理
- `stage advance` / `phase complete` 実行時に、対応する `processes` のステータスも自動更新

#### A-3. 自動検出可能な完了条件

| 検出タイプ | 実装方法 | Phase |
|-----------|---------|-------|
| `issue_label` | GitHub API: ラベル付きIssueのopen/closed比率 | Phase2 |
| `pr_merged` | GitHub API: 特定ラベルPRのマージ状態 | Phase2 |
| `document_exists` | ローカルファイル存在チェック | Phase1 |
| `test_pass_rate` | `go test` / CI結果のパース | Phase1(ローカル), Phase2(Actions) |
| `coverage` | カバレッジレポートのパース | Phase1(ローカル), Phase2(Actions) |
| `issue_ratio` | GitHub API: ラベル別Issue完了率 | Phase2 |
| `manual_approval` | `teraflow gate approve <process>` コマンド | Phase1 |

Phase1ではローカル検出（ファイル存在、テスト結果）+ 手動承認。
Phase2でGitHub API連携による自動検出を追加。

### B. ロール別「次にやるべきこと」表示の設計案

#### B-1. 想定ロール定義

| ロール | 英語キー | 主な関心事 |
|--------|---------|-----------|
| プロジェクトマネージャー | `pm` | 全体進捗、ゲート承認、リスク |
| 開発者 | `dev` | 現在のフェーズのタスク、実装方針 |
| QA・テスター | `qa` | テスト計画、品質メトリクス、不具合状況 |
| ステークホルダー | `stakeholder` | マイルストーン、成果物の状態 |

#### B-2. コマンドIF案

```bash
# ロール別次アクション表示
teraflow status --role=pm
teraflow status --role=dev
teraflow status --role=qa
teraflow status --role=stakeholder

# ロール未指定時はデフォルト（全ロールのサマリ）
teraflow status
```

#### B-3. 表示イメージ（ASCIIモックアップ）

**PM向け (`teraflow status --role=pm`)**:
```
╔═══════════════════════════════════════════════════════╗
║  teraflow Status — PM View                            ║
╠═══════════════════════════════════════════════════════╣
║  Project: my-saas-app                                 ║
║  Stage:   initial_development                         ║
║  Phase:   detailed_design (3/6)                       ║
╠═══════════════════════════════════════════════════════╣
║                                                       ║
║  📋 次にやるべきこと (PM)                              ║
║  ─────────────────────────────────────────────        ║
║  1. [要承認] 詳細設計レビューのゲート承認              ║
║     → teraflow gate approve detailed_design           ║
║     条件: Issue完了率 92% (閾値90% ✅)                 ║
║           手動承認: 未 ❌                              ║
║                                                       ║
║  2. [確認] リワーク2件が未解決                         ║
║     → teraflow rework list --status=open              ║
║                                                       ║
║  3. [計画] 次フェーズ「実装」の開始準備                ║
║     → 開発者のアサイン確認                             ║
║     → テスト計画の事前レビュー                         ║
║                                                       ║
║  ⚠️ リスク                                            ║
║  - 詳細設計の手戻り2件 → 実装開始に影響の可能性       ║
║  - 完了予定日: 2026-03-10 (残5日)                     ║
╚═══════════════════════════════════════════════════════╝
```

**開発者向け (`teraflow status --role=dev`)**:
```
╔═══════════════════════════════════════════════════════╗
║  teraflow Status — Developer View                     ║
╠═══════════════════════════════════════════════════════╣
║  Phase: detailed_design (進行中)                      ║
║  対応する共通フレーム: 2.4.4 ソフトウェア詳細設計      ║
╠═══════════════════════════════════════════════════════╣
║                                                       ║
║  📋 次にやるべきこと (Developer)                       ║
║  ─────────────────────────────────────────────        ║
║  1. [作業] 未完了の設計Issue 3件                       ║
║     → gh issue list --label detailed-design --state open ║
║                                                       ║
║  2. [対応] リワーク指摘への修正                        ║
║     → RW-005: API設計の整合性 (target: detailed_design)║
║     → RW-006: DB設計のインデックス不足                 ║
║                                                       ║
║  3. [次の準備] 実装フェーズの事前確認                  ║
║     → 開発環境セットアップ確認: teraflow doctor        ║
║                                                       ║
║  📊 進捗                                              ║
║  設計Issue: 12/15 完了 (80%)                          ║
║  設計レビューPR: 5/6 マージ済み                        ║
╚═══════════════════════════════════════════════════════╝
```

**QA向け (`teraflow status --role=qa`)**:
```
╔═══════════════════════════════════════════════════════╗
║  teraflow Status — QA View                            ║
╠═══════════════════════════════════════════════════════╣
║  Phase: detailed_design (テストフェーズ前)             ║
╠═══════════════════════════════════════════════════════╣
║                                                       ║
║  📋 次にやるべきこと (QA)                              ║
║  ─────────────────────────────────────────────        ║
║  1. [準備] テスト計画書の作成・更新                    ║
║     → 現在の設計Issueからテストケース抽出              ║
║                                                       ║
║  2. [監視] 品質メトリクス                              ║
║     → 手戻り率: 2/15 (13.3%)                          ║
║     → 障害チケット: 0件 (フェーズ前のため正常)         ║
║                                                       ║
║  3. [レビュー] 設計レビューへの参加                    ║
║     → 未レビューPR: 1件                               ║
║                                                       ║
║  📊 品質指標                                          ║
║  ハーネススコア: 78/100                                ║
║  リワーク発生率: 13.3% (閾値20%以下 ✅)               ║
╚═══════════════════════════════════════════════════════╝
```

**ステークホルダー向け (`teraflow status --role=stakeholder`)**:
```
╔═══════════════════════════════════════════════════════╗
║  teraflow Status — Stakeholder View                   ║
╠═══════════════════════════════════════════════════════╣
║  Project: my-saas-app                                 ║
║  全体進捗: 50% (3/6フェーズ完了)                      ║
╠═══════════════════════════════════════════════════════╣
║                                                       ║
║  📋 状況サマリ                                        ║
║  ─────────────────────────────────────────────        ║
║  ✅ 企画         → 完了 (2026-01-20)                  ║
║  ✅ 要件定義     → 完了 (2026-02-05)                  ║
║  ✅ 基本設計     → 完了 (2026-02-20)                  ║
║  🔄 詳細設計     → 進行中 (完了予定: 2026-03-10)      ║
║  ⏳ 実装         → 未着手                             ║
║  ⏳ テスト       → 未着手                             ║
║                                                       ║
║  📋 次のマイルストーン                                ║
║  → 詳細設計完了レビュー (2026-03-10予定)              ║
║  → 実装フェーズ開始 (2026-03-11予定)                  ║
║                                                       ║
║  ⚠️ 注意事項                                         ║
║  手戻り2件が設計フェーズで発生中                       ║
╚═══════════════════════════════════════════════════════╝
```

#### B-4. ロール別表示のデータソース

| データ | Phase1（ローカル） | Phase2（GitHub連携） |
|--------|-------------------|---------------------|
| プロセス状態 | project-state.yml | project-state.yml + GitHub API |
| 未完了タスク | ローカルIssueリスト不可→手動 | GitHub Issues API |
| リワーク状況 | rework-log.yml | rework-log.yml + Issue連携 |
| テスト結果 | `go test` 結果パース | Actions ワークフロー結果 |
| PR状態 | 不可→省略 | GitHub PR API |
| カバレッジ | `coverage.out` パース | Actions アーティファクト |

### C. プロセス全体可視化の設計案

#### C-1. `teraflow process` コマンド

```bash
# プロセス全体の進捗表示
teraflow process

# 特定プロセスの詳細
teraflow process show <name>

# ゲート条件の確認
teraflow process gate <name>

# 手動ゲート承認
teraflow process approve <name>
```

#### C-2. 表示イメージ

**`teraflow process` 実行結果:**
```
╔═══════════════════════════════════════════════════════════════════╗
║  teraflow Process Overview — my-saas-app                         ║
╠═══════════════════════════════════════════════════════════════════╣
║                                                                   ║
║  企画 ✅ ─→ 要件定義 ✅ ─→ 基本設計 ✅ ─→ 詳細設計 🔄 ─→ 実装 ⏳ ─→ テスト ⏳ ─→ リリース ⏳ ║
║                                                ↑ 現在ここ          ║
║                                                Phase 4/7          ║
║                                                                   ║
╠═══════════════════════════════════════════════════════════════════╣
║  プロセス         │ 状態    │ SLCP-JCF   │ 開始日     │ 完了日     ║
║  ─────────────────┼─────────┼────────────┼────────────┼────────── ║
║  企画             │ ✅ 完了  │ 2.1        │ 2026-01-15 │ 2026-01-20║
║  要件定義         │ ✅ 完了  │ 2.2        │ 2026-01-21 │ 2026-02-05║
║  基本設計         │ ✅ 完了  │ 2.4.3      │ 2026-02-06 │ 2026-02-20║
║  詳細設計         │ 🔄 進行中│ 2.4.4      │ 2026-02-21 │ —         ║
║  実装             │ ⏳ 未着手│ 2.4.5      │ —          │ —         ║
║  テスト           │ ⏳ 未着手│ 2.4.6-7    │ —          │ —         ║
║  リリース         │ ⏳ 未着手│ 2.3.7-8    │ —          │ —         ║
╠═══════════════════════════════════════════════════════════════════╣
║                                                                   ║
║  🚪 次のゲート: 詳細設計レビュー                                  ║
║  ─────────────────────────────────────────────                    ║
║  条件1: Issue完了率 ≥ 90%     → 92% ✅                            ║
║  条件2: PM手動承認            → 未承認 ❌                          ║
║  → teraflow process approve detailed_design                       ║
║                                                                   ║
║  📊 全体進捗: ████████████░░░░░░░░  43% (3/7プロセス完了)         ║
║  📈 手戻り率: 13.3% (閾値20%以下 ✅)                              ║
╚═══════════════════════════════════════════════════════════════════╝
```

**`teraflow process show detailed_design` 実行結果:**
```
╔═══════════════════════════════════════════════════════╗
║  Process: 詳細設計 (detailed_design)                  ║
║  SLCP-JCF: 2.4.4 ソフトウェア詳細設計                ║
╠═══════════════════════════════════════════════════════╣
║  状態:    🔄 進行中                                   ║
║  開始日:  2026-02-21                                  ║
║  予定日:  2026-03-10                                  ║
║  残日数:  5日                                         ║
╠═══════════════════════════════════════════════════════╣
║                                                       ║
║  ゲート条件                                           ║
║  ─────────────────────────────────────────────        ║
║  [✅] Issue完了率 ≥ 90%     (現在: 92%)               ║
║  [❌] PM手動承認             (未承認)                  ║
║                                                       ║
║  関連リワーク                                         ║
║  ─────────────────────────────────────────────        ║
║  RW-005: API設計の整合性 (open)                       ║
║  RW-006: DB設計のインデックス不足 (open)              ║
║                                                       ║
║  共通フレーム アクティビティ対応                       ║
║  ─────────────────────────────────────────────        ║
║  2.4.4.1 プログラム詳細設計                           ║
║  2.4.4.2 データベース詳細設計                         ║
║  2.4.4.3 ソフトウェアユニットテスト設計               ║
╚═══════════════════════════════════════════════════════╝
```

#### C-3. JSON出力対応

既存の `--output=json` フラグとの一貫性を保ち、全コマンドでJSON出力に対応:

```bash
teraflow process --output=json
teraflow status --role=pm --output=json
```

Phase2のGitHub Actions連携時に、JSON出力を他ツール（GitHub Projects、Slack通知等）に連携可能。

### D. 実装難易度・工数見積もり

#### D-1. 機能別見積もり

| 機能 | 難易度 | 規模 | 変更対象ファイル | 既存への影響 | 優先度 |
|------|--------|------|----------------|------------|--------|
| **A. プロセス状態追跡** | | | | | |
| A-1. project-state.yml拡張 | 中 | M | `internal/state/state.go`, `internal/state/process.go`(新規) | SaveState/LoadState拡張（後方互換） | P0 |
| A-2. gate_rules定義・評価 | 高 | L | `internal/gate/`(新規パッケージ) | なし（新規） | P1 |
| A-3. stage/phase連動更新 | 中 | M | `cmd/stage.go`, `cmd/phase.go` | advance/complete時にprocess更新 | P0 |
| **B. ロール別表示** | | | | | |
| B-1. status --role フラグ | 低 | S | `cmd/status.go`(新規) | なし（新規コマンド） | P1 |
| B-2. ロール別テンプレート | 中 | M | `internal/display/role_view.go`(新規) | なし（新規） | P1 |
| B-3. データ集約ロジック | 中 | M | `internal/display/aggregator.go`(新規) | LoadState等を呼び出し | P1 |
| **C. プロセス可視化** | | | | | |
| C-1. process コマンド | 低 | S | `cmd/process.go`(新規) | なし（新規コマンド） | P0 |
| C-2. プログレスバー/テーブル表示 | 低 | S | `internal/display/progress.go`(新規) | なし（新規） | P0 |
| C-3. gate条件表示 | 中 | M | `cmd/process.go` に統合 | A-2に依存 | P1 |

#### D-2. 合計工数見積もり

| カテゴリ | タスク数 | 規模内訳 | 推定セッション数 |
|---------|---------|---------|---------------|
| A. プロセス状態追跡 | 3 | S×0, M×2, L×1 | 3〜4 |
| B. ロール別表示 | 3 | S×1, M×2 | 2〜3 |
| C. プロセス可視化 | 3 | S×2, M×1 | 2 |
| テスト | 1 | M | 1〜2 |
| **合計** | **10** | **S×3, M×5, L×1** | **8〜11** |

#### D-3. 推奨実装順序

```
Wave 1 (P0 — 基盤):
  A-1. project-state.yml 拡張 (M)
  A-3. stage/phase連動 (M)
  C-1. process コマンド基本形 (S)
  C-2. プログレスバー表示 (S)

Wave 2 (P1 — UX向上):
  B-1. status --role フラグ (S)
  B-2. ロール別テンプレート (M)
  B-3. データ集約 (M)
  C-3. gate条件表示 (M)

Wave 3 (P1 — 自動化):
  A-2. gate_rules 自動評価 (L)
  テスト (M)
```

#### D-4. Phase1/Phase2の分界

| 機能 | Phase1（ローカルCLI） | Phase2（GitHub連携） |
|------|---------------------|---------------------|
| プロセス状態表示 | ✅ project-state.ymlから表示 | ✅ + Issue/PR状態を自動反映 |
| ロール別表示 | ✅ ローカルデータのみ | ✅ + GitHub APIからデータ取得 |
| ゲート自動評価 | △ ファイル存在+手動承認のみ | ✅ Issue完了率、PR状態、CI結果 |
| 次アクション表示 | ✅ 静的ルールベース | ✅ + 動的ルール（Issueベース） |

Phase1ではローカルデータ（project-state.yml, rework-log.yml）ベースの表示に限定し、Phase2でGitHub API連携による自動検出・動的表示を追加する。この段階的アプローチにより、Phase1の実装をブロックせずに殿の要件を満たせる。

---

## 8. 権限管理・RBAC設計案（Step 0b追加要件2）

### 背景

共通フレームのプロセス管理を導入する以上、**誰がどのプロセスを実行・承認できるか**の権限制御が不可欠。以下5つの観点で設計案を提示する。

### E. RBAC（ロールベースアクセス制御）

#### E-1. 共通フレームに基づくロール定義

| ロール | 英語キー | 共通フレーム対応 | 主な権限 |
|--------|---------|----------------|---------|
| プロジェクトマネージャー | `pm` | 5.1 プロジェクト計画 | ゲート承認、ステージ遷移、リスク管理 |
| アーキテクト | `architect` | 2.3.3 システム方式設計 | 設計承認、技術判断 |
| 開発者 | `developer` | 2.4.5 ソフトウェア構築 | フェーズ内作業、PR作成 |
| QAエンジニア | `qa` | 4.3 検証 / 4.4 妥当性確認 | テスト実行、品質判定 |
| リリースマネージャー | `release_mgr` | 2.3.7 システム導入 | リリース承認、デプロイ |
| ステークホルダー | `stakeholder` | 1.1 取得プロセス | 閲覧のみ、受入れ承認 |

#### E-2. teraflow.yml でのRBAC設定

```yaml
# .github/teraflow.yml
rbac:
  enabled: true
  
  roles:
    pm:
      members:
        - github: "taka-sho"         # GitHubユーザー名
        - team: "project-leads"       # GitHub Team
      permissions:
        - "gate.approve.*"            # 全ゲート承認可
        - "stage.advance"             # ステージ遷移可
        - "process.approve.*"         # 全プロセス承認可
        - "rework.approve"            # 手戻り承認
        - "*.read"                    # 全情報閲覧
    
    architect:
      members:
        - team: "architects"
      permissions:
        - "gate.approve.basic_design"
        - "gate.approve.detailed_design"
        - "process.approve.basic_design"
        - "process.approve.detailed_design"
        - "*.read"
    
    developer:
      members:
        - team: "developers"
      permissions:
        - "phase.start"              # フェーズ内作業開始
        - "rework.create"            # 手戻り申請
        - "incident.create"          # 障害報告
        - "*.read"
    
    qa:
      members:
        - team: "qa-team"
      permissions:
        - "gate.approve.testing"
        - "gate.approve.integration_test"
        - "incident.create"
        - "incident.close"
        - "*.read"
    
    release_mgr:
      members:
        - team: "release-team"
      permissions:
        - "gate.approve.release"
        - "stage.advance"            # release → operation 遷移
        - "*.read"
    
    stakeholder:
      members:
        - team: "stakeholders"
      permissions:
        - "*.read"                   # 閲覧のみ
```

#### E-3. 権限チェックの実装方式

**Phase1（ローカルCLI）**: teraflow.yml の `rbac` セクションを読み込み、`git config user.name` または `--user` フラグで現在のユーザーを特定。権限不足時はエラーメッセージで拒否。

```go
// internal/rbac/rbac.go (新規)
type Permission struct {
    Resource string // "gate", "stage", "phase", "rework", "incident", "process"
    Action   string // "approve", "advance", "start", "create", "read"
    Target   string // "basic_design", "testing", "*"
}

func CheckPermission(config *Config, user string, perm Permission) error {
    role := resolveRole(config, user)
    if !role.HasPermission(perm) {
        return fmt.Errorf("permission denied: %s requires %s.%s.%s (your role: %s)",
            user, perm.Resource, perm.Action, perm.Target, role.Name)
    }
    return nil
}
```

**Phase2（GitHub連携）**: GitHub Teams API + CODEOWNERS連携で動的にロール解決。Branch Protection Rulesとの統合で二重チェック。

### F. ゲート承認権限（フェーズ遷移制御）

#### F-1. 承認フロー

```
開発者: teraflow phase complete
  → 権限チェック: developer は phase.complete 不可（ゲート承認が必要なフェーズ）
  → エラー: "詳細設計の完了にはPMまたはアーキテクトの承認が必要です"
  → teraflow gate approve detailed_design

PM: teraflow gate approve detailed_design
  → 権限チェック: pm は gate.approve.detailed_design ✅
  → ゲート条件チェック: Issue完了率92% ✅, テスト未実行（設計フェーズ）→ スキップ
  → 承認記録: audit-log.yml に記録
  → フェーズ自動遷移: detailed_design → implementation
```

#### F-2. ゲート承認要件マトリクス

| フェーズ遷移 | 必要ロール | ゲート条件 | 強制制約 |
|------------|-----------|-----------|---------|
| 企画 → 要件定義 | pm | 企画書承認 | なし |
| 要件定義 → 基本設計 | pm | 要件仕様書承認、SH合意 | なし |
| 基本設計 → 詳細設計 | pm or architect | 設計レビュー完了 | レビューPRマージ済み |
| 詳細設計 → 実装 | pm or architect | 詳細設計レビュー完了 | Issue完了率≥90% |
| 実装 → テスト | pm | コードレビュー完了 | ユニットテスト全件PASS |
| テスト → リリース | pm and qa | 結合テスト完了 | テスト通過率100%, カバレッジ≥80% |
| リリース → 運用 | release_mgr | 受入テスト完了 | PM+SH承認済み |

#### F-3. 複数承認者対応

```yaml
gate_rules:
  testing_to_release:
    approvers:
      - role: pm
        required: true     # 必須
      - role: qa
        required: true     # 必須
    quorum: all            # all | majority | any
```

### G. 操作の強制制約（システムブロック）

#### G-1. 制約ルール定義

```yaml
# .github/teraflow.yml
constraints:
  # テスト未通過でリリースフェーズに遷移不可
  - id: C001
    trigger: "phase.advance.release"
    condition: "gate.testing.test_pass_rate < 1.0"
    action: block
    message: "テスト通過率が100%未満です。リリースフェーズに遷移できません。"
  
  # カバレッジ不足でリリース不可
  - id: C002
    trigger: "phase.advance.release"
    condition: "gate.testing.coverage < 0.80"
    action: block
    message: "カバレッジが80%未満です。(現在: {{coverage}}%)"
  
  # 未解決リワークがある状態でステージ遷移不可
  - id: C003
    trigger: "stage.advance"
    condition: "rework.open_count > 0"
    action: warn           # warn | block
    message: "未解決の手戻りが{{open_count}}件あります。"
  
  # 未解決障害がある状態でリリース不可
  - id: C004
    trigger: "phase.advance.release"
    condition: "incident.open_count > 0 AND incident.max_severity == 'critical'"
    action: block
    message: "未解決のクリティカル障害があります。リリースできません。"
  
  # 設計書が存在しない状態で実装フェーズに遷移不可
  - id: C005
    trigger: "phase.advance.implementation"
    condition: "!file_exists('docs/design/*.md')"
    action: block
    message: "設計書が作成されていません。"
```

#### G-2. 制約エンジン

```go
// internal/constraint/engine.go (新規)
type Constraint struct {
    ID        string
    Trigger   string    // "phase.advance.release" etc.
    Condition string    // 式言語で評価
    Action    string    // "block" or "warn"
    Message   string
}

type Engine struct {
    constraints []Constraint
    evaluator   ConditionEvaluator
}

func (e *Engine) Check(trigger string, ctx *EvalContext) ([]Violation, error) {
    // trigger にマッチする制約を評価し、違反リストを返す
}
```

Phase1では条件式を簡易的に実装（Go内蔵）。Phase2ではGitHub APIデータを評価コンテキストに追加。

#### G-3. `--force` フラグ（緊急時のオーバーライド）

```bash
# 通常: ブロックされる
teraflow stage advance
# Error: 未解決の手戻りが2件あります (C003)

# 緊急時: PMロール + --force で制約を上書き（監査ログに記録）
teraflow stage advance --force --reason="ビジネス要件により緊急リリース"
# Warning: 制約C003をオーバーライドしました。監査ログに記録済み。
```

`--force` の実行にはpmまたはrelease_mgrロールが必要。使用時は監査ログに理由が必須記録される。

### H. GitHub権限統合

#### H-1. 連携対象

| GitHub機能 | teraflow連携方法 | Phase |
|-----------|-----------------|-------|
| **CODEOWNERS** | ファイルパスベースのレビュー必須化をゲート条件に統合 | Phase2 |
| **GitHub Teams** | RBACのmembers解決に使用 (`team: "developers"`) | Phase2 |
| **Branch Protection** | ゲート承認済みのブランチのみマージ可能 | Phase2 |
| **Required Reviews** | 設計レビュー/コードレビューの承認状態をゲート条件に統合 | Phase2 |
| **Status Checks** | CI/テスト結果をゲート条件の自動評価に使用 | Phase2 |

#### H-2. Phase1での代替手段

Phase1（ローカルCLI）ではGitHub APIを直接使えないため:
- CODEOWNERS → teraflow.yml の rbac.roles で代替
- Teams → teraflow.yml の members リストで代替
- Branch Protection → `teraflow gate approve` の手動承認で代替
- Required Reviews → `teraflow gate` の手動チェックで代替
- Status Checks → `teraflow doctor` + `go test` 結果で代替

### I. 監査ログ

#### I-1. 監査ログの構造

```yaml
# .teraflow/audit-log.yml
entries:
  - id: "AUD-001"
    timestamp: "2026-03-10T14:30:00+09:00"
    user: "taka-sho"
    role: "pm"
    action: "gate.approve"
    target: "detailed_design"
    result: "approved"
    conditions_met:
      - "issue_ratio: 0.92 (threshold: 0.90)"
    note: ""
  
  - id: "AUD-002"
    timestamp: "2026-03-15T09:00:00+09:00"
    user: "taka-sho"
    role: "pm"
    action: "stage.advance"
    target: "initial_development → release"
    result: "approved"
    constraints_overridden:
      - id: "C003"
        reason: "ビジネス要件により緊急リリース"
    note: "取締役会報告期限のため"
  
  - id: "AUD-003"
    timestamp: "2026-03-15T09:01:00+09:00"
    user: "dev-user"
    role: "developer"
    action: "stage.advance"
    target: "initial_development → release"
    result: "denied"
    reason: "permission denied: developer cannot stage.advance"
```

#### I-2. 監査ログコマンド

```bash
# 監査ログ表示
teraflow audit list
teraflow audit list --user=taka-sho
teraflow audit list --action=gate.approve
teraflow audit list --since=2026-03-01

# JSON出力（外部ツール連携用）
teraflow audit list --output=json
```

#### I-3. 監査ログのPhase1/Phase2対応

| 機能 | Phase1 | Phase2 |
|------|--------|--------|
| ログ保存先 | `.teraflow/audit-log.yml`（ローカル） | + GitHub Actions Artifactsへのアップロード |
| ユーザー識別 | `git config user.name` | GitHub API（認証済みユーザー） |
| ログ検索 | `teraflow audit list` | + GitHub ActionsログUI連携 |
| 改ざん防止 | なし（ローカルファイル） | gitコミット署名 + Branch Protection |

### D. 実装難易度・工数見積もり（改訂版: RBAC含む）

#### D-1. 機能別見積もり（改訂）

| 機能 | 難易度 | 規模 | 変更対象ファイル | 既存への影響 | 優先度 |
|------|--------|------|----------------|------------|--------|
| **A. プロセス状態追跡** | | | | | |
| A-1. project-state.yml拡張 | 中 | M | `internal/state/state.go`, `internal/state/process.go`(新規) | SaveState/LoadState拡張 | P0 |
| A-2. gate_rules定義・評価 | 高 | L | `internal/gate/`(新規) | なし | P1 |
| A-3. stage/phase連動更新 | 中 | M | `cmd/stage.go`, `cmd/phase.go` | advance/complete拡張 | P0 |
| **B. ロール別表示** | | | | | |
| B-1. status --role フラグ | 低 | S | `cmd/status.go`(新規) | なし | P1 |
| B-2. ロール別テンプレート | 中 | M | `internal/display/`(新規) | なし | P1 |
| B-3. データ集約ロジック | 中 | M | `internal/display/`(新規) | なし | P1 |
| **C. プロセス可視化** | | | | | |
| C-1. process コマンド | 低 | S | `cmd/process.go`(新規) | なし | P0 |
| C-2. プログレスバー表示 | 低 | S | `internal/display/`(新規) | なし | P0 |
| C-3. gate条件表示 | 中 | M | C-1に統合 | A-2に依存 | P1 |
| **E. RBAC** | | | | | |
| E-1. RBACエンジン | 高 | L | `internal/rbac/`(新規パッケージ) | なし | P0 |
| E-2. teraflow.yml rbacセクション | 中 | M | `internal/config/`拡張 | config読み込み拡張 | P0 |
| E-3. 全コマンドへの権限チェック注入 | 高 | L | `cmd/*.go` 全般 | 既存コマンドにチェック追加 | P1 |
| **F. ゲート承認権限** | | | | | |
| F-1. gate approveコマンド | 中 | M | `cmd/gate.go`(新規) | なし | P0 |
| F-2. 複数承認者対応 | 高 | M | `internal/gate/`拡張 | A-2に依存 | P2 |
| **G. 強制制約** | | | | | |
| G-1. 制約エンジン | 高 | L | `internal/constraint/`(新規) | なし | P1 |
| G-2. --forceオーバーライド | 中 | S | 各cmd統合 | G-1に依存 | P2 |
| **H. GitHub権限統合** | | | | | |
| H-1. Teams/CODEOWNERS連携 | 高 | L | `internal/github/`拡張 | Phase2のみ | P2 |
| **I. 監査ログ** | | | | | |
| I-1. audit-log.yml書き込み | 低 | S | `internal/audit/`(新規) | なし | P0 |
| I-2. audit コマンド | 低 | S | `cmd/audit.go`(新規) | なし | P1 |

#### D-2. 合計工数見積もり（改訂）

| カテゴリ | タスク数 | 規模内訳 | 推定セッション数 |
|---------|---------|---------|---------------|
| A. プロセス状態追跡 | 3 | M×2, L×1 | 3〜4 |
| B. ロール別表示 | 3 | S×1, M×2 | 2〜3 |
| C. プロセス可視化 | 3 | S×2, M×1 | 2 |
| E. RBAC | 3 | M×1, L×2 | 4〜5 |
| F. ゲート承認 | 2 | M×2 | 2 |
| G. 強制制約 | 2 | S×1, L×1 | 2〜3 |
| H. GitHub権限統合 | 1 | L×1 | 2〜3 |
| I. 監査ログ | 2 | S×2 | 1 |
| テスト | 2 | M×2 | 2〜3 |
| **合計** | **21** | **S×6, M×8, L×5** | **20〜26** |

#### D-3. 推奨実装順序（改訂）

```
Wave 1 (P0 — 基盤): 6〜8セッション
  A-1. project-state.yml 拡張 (M)
  A-3. stage/phase連動 (M)
  C-1. process コマンド基本形 (S)
  C-2. プログレスバー表示 (S)
  E-1. RBACエンジン (L)
  E-2. teraflow.yml rbac設定 (M)
  F-1. gate approveコマンド (M)
  I-1. audit-log.yml書き込み (S)

Wave 2 (P1 — UX+権限): 8〜10セッション
  B-1. status --role フラグ (S)
  B-2. ロール別テンプレート (M)
  B-3. データ集約 (M)
  C-3. gate条件表示 (M)
  E-3. 全コマンド権限チェック (L)
  A-2. gate_rules自動評価 (L)
  G-1. 制約エンジン (L)
  I-2. audit コマンド (S)

Wave 3 (P2 — 高度機能): 4〜6セッション
  F-2. 複数承認者対応 (M)
  G-2. --force オーバーライド (S)
  H-1. GitHub Teams/CODEOWNERS連携 (L)
  テスト (M×2)
```

#### D-4. 重要な設計判断ポイント

1. **RBAC粒度**: 共通フレームのプロセス単位でパーミッションを定義する。過剰な粒度（アクティビティ単位）は設定複雑化を招くため非推奨。
2. **Phase1のRBAC**: ローカルCLIではGitHub認証不可のため、`git config user.name` + teraflow.yml の members リストで簡易認証。セキュリティは低いが、プロセス教育ツールとしては十分。
3. **Phase2でのセキュリティ強化**: GitHub App認証 + Teams API で正式なRBACを実現。Phase1のteraflow.yml設定はPhase2でも引き続き有効（フォールバック）。
4. **監査ログの改ざん防止**: Phase1はローカルYAMLのため改ざん可能。Phase2でgitコミット署名による完全性保証を追加。
