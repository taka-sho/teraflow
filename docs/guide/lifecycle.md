---
codd:
  node_id: "docs:lifecycle"
  title: "teraflow ライフサイクル詳細"
  depends_on:
    - id: "req:teraflow-overview"
      relation: implements
---

# teraflow ライフサイクル詳細

## 1. ライフサイクル全体図

```text
[initial_development] → [release] → [operation]
                                        ↓
                                  [maintenance]
                                        ↓
                             [continuous_improvement] → [retirement]
```

実運用では `operation` と `continuous_improvement` が並行して進む。`maintenance` は必要時に有効化される補助ステージとして扱う。

## 2. 各ステージの説明

- `initial_development`: 初回リリースまでの主要開発。要求整理からテストまでを順序立てて実施する。
- `release`: 移行計画、環境準備、データ移行、本番切替、安定化確認を行う。
- `operation`: 稼働中システムの障害対応・問い合わせ対応をイベント駆動で処理する。
- `continuous_improvement`: リリース後の機能追加・改善。短いサイクルを反復し、必要に応じてフェーズをスキップする。
- `maintenance`: 技術的負債返済、ライブラリ更新、EOL対応などを扱う。
- `retirement`: サービス終了時の移行、告知、停止、アーカイブを管理する。

## 3. フェーズ遷移図

```text
requirements
  → basic_design
  → detailed_design
  → implementation
  → testing
  → integration_test
```

フェーズ遷移は `teraflow phase` 系コマンドで管理し、履歴を `project-state.yml` に記録する。

## 4. ゲート条件の仕組み

ゲート条件は `teraflow.yml` の `gates` / `phases` セクションに定義する。代表例は以下。

- 必須成果物ファイルの存在
- 計画成果物数と確定成果物数の一致
- ゲートIssueのクローズ状態
- ステージ固有の追加チェック（例: リリース判定）

`teraflow stage advance` や `teraflow phase advance` 実行時にゲート判定を行い、未達なら遷移を拒否する。

## 5. 手戻りフロー

```text
rework create
  → 影響分析（CoDD）
  → 承認
  → phase reversion（フェーズ後退）
  → 修正
  → 再度 advance で復帰
```

手戻りは `teraflow rework create` で起票し、承認後にフェーズを正式に戻す。再実行ポリシー（full / impact_based / manual）に従い、必要範囲を再通過して解消する。
