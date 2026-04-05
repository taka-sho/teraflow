---
codd:
  node_id: "adr:010-slcp-jcf-partial-migration"
  title: "ADR-010: SLCP-JCF（共通フレーム）部分移行"
  depends_on:
    - id: "req:slcp-jcf-compliance"
      relation: implements
    - id: "adr:004-data-github"
      relation: extends
    - id: "adr:008-phase1-coexistence"
      relation: refines
---

# ADR-010: SLCP-JCF（共通フレーム）部分移行

## ステータス

承認済み（Accepted）

## コンテキスト

teraflowはTerasoluna開発プロセスフレームワークを基盤としてステージ/フェーズ概念を設計していた。公共調達・企業導入においてIPA共通フレーム（SLCP-JCF2013, ISO/IEC 12207:2008準拠）への対応が求められるようになった。

### 検討の契機

- 公共調達要件での「共通フレーム準拠」要求への対応
- 利用者（PJチーム全員）がプロセスの現在状態を把握し、次に何をすべきかを判断できるUXの実現
- ロールベースのアクセス制御によるガバナンス強化

### 制約

- Phase2の16ワークフロー（TF-001〜016）がstage/phase概念に依存しており、破壊的変更は高コスト
- teraflow独自概念（rework, harness score, group, CoDD）は共通フレームに対応がなく、これらは競争力の源泉
- 既存ユーザーのCLI操作を壊してはならない

## 決定

**案B（部分移行 + プロセス可視化 + RBAC）を採用する。**

### 案B の内容

1. **ドキュメント層のみ更新**: docs/内のTerasoluna参照を共通フレーム参照に置換。SLCP-JCF対応表を新規作成
2. **CLI変更なし**: `teraflow stage`, `teraflow phase`等の既存コマンド名を維持
3. **新機能追加**:
   - `teraflow process`: プロセス全体の可視化（状態追跡、プログレスバー、SLCP-JCF番号併記）
   - `teraflow status --role=<role>`: ロール別「次にやるべきこと」表示（PM/dev/QA/stakeholder）
   - RBAC権限管理: teraflow.yml rbacセクション、ゲート承認権限、操作の強制制約
   - 監査ログ: `.teraflow/audit-log.yml`

## 却下した案

### 案A: 完全移行（Terasoluna概念を廃止し共通フレームに統一）

CLIコマンド名を`stage`→`process`、`phase`→`activity`に変更し、全ドキュメント・全ソースコード・全テストを改修する案。

**却下理由:**
- Phase2の16ワークフローが全面改修となり、工数Lで見合わない
- teraflow独自概念（rework, harness score）を共通フレーム用語に無理に置き換えると差別化を失う
- 既存ユーザーへの破壊的変更。テスト全面書き直し
- ISO/IEC 12207はテーラリングを前提としており、コマンド名の完全一致は規格上不要

### 案C: エイリアス方式（共通フレーム用語を追加し既存コマンドもエイリアスとして維持）

案Bのドキュメント更新に加え、`teraflow process advance` = `teraflow stage advance`（両方動作）のエイリアスを追加する案。

**却下理由:**
- コマンド体系が2倍になりメンテナンスコスト増
- ドキュメントでどちらの用語を主にするか判断が必要で混乱を招く
- 案Bで対応表を確立した後に案Cへの移行は容易であり、現時点では過剰

## 結果

### 変更される点

1. ドキュメント層: 全docs/内のTerasoluna参照→共通フレーム参照に更新（11ファイル + 新規1ファイル）
2. 新コマンド: `teraflow process`, `teraflow status --role`, `teraflow gate approve`, `teraflow audit list`
3. 状態管理: project-state.ymlにprocessesセクション追加
4. 設定: teraflow.ymlにrbac, gate_rules, constraintsセクション追加
5. ログ: `.teraflow/audit-log.yml` 新設

### 変更されない点

1. CLIコマンド名: stage, phase, rework, incident, harness等は全て維持
2. 既存コマンドの挙動: 全て後方互換
3. project-state.ymlの既存フィールド: lifecycle.current_stage, phases.current は維持
4. teraflow独自概念: rework, harness score, group, CoDD は変更なし

### 工数見積もり

| カテゴリ | タスク数 | 規模 | セッション数 |
|---------|---------|------|------------|
| ドキュメント層更新 | 12 | S〜M | 3〜4 |
| プロセス可視化 | 3 | S〜L | 3〜4 |
| ロール別表示 | 3 | S〜M | 2〜3 |
| RBAC | 3 | M〜L | 4〜5 |
| ゲート承認 | 2 | M | 2 |
| 強制制約 | 2 | S〜L | 2〜3 |
| 監査ログ | 2 | S | 1 |
| テスト | 2 | M | 2〜3 |
| **合計** | **29** | -- | **19〜25** |

### リスク

- Phase1のRBACは`git config user.name`ベースで容易に回避可能（セキュリティとしては不十分だが、プロセス教育ツールとしては許容）
- SLCP-JCF2013は2013年版であり、次期改訂時に対応表の更新が必要
- 21タスク規模の実装がPhase2開発と並行するため、リソース競合の可能性

### 関連文書

- docs/design/analysis-slcp-jcf-migration.md（Step 0調査・分析）
- docs/requirements/req-slcp-jcf-compliance.md（本ADRに基づく要件定義）
