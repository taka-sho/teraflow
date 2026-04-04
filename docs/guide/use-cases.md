---
codd:
  node_id: "docs:use-cases"
  title: "teraflow ユースケース集"
  depends_on:
    - id: "req:teraflow-overview"
      relation: implements
---

# teraflow ユースケース集

## シナリオ1: 新規プロジェクト立ち上げ

```bash
teraflow init --name "注文管理システム" --non-interactive
teraflow status
teraflow config set project.repository "https://github.com/org/order-system"
teraflow label sync
```

プロジェクト雛形を生成し、リポジトリ設定とラベル体系を初期化する。

## シナリオ2: フェーズ進捗管理

```bash
teraflow phase list              # 現在フェーズ確認
teraflow phase complete          # 要件定義完了→基本設計へ
teraflow status                  # 状態確認
teraflow changelog add feat "要件定義完了"
```

日次運用でフェーズ状態を確認し、完了時に履歴を残す。

## シナリオ3: 手戻り発生時

```bash
teraflow rework create --group group-auth --target-phase basic_design --reason "API仕様変更"
teraflow rework list
teraflow dashboard show          # 全体状況確認
```

手戻りを起票し、影響範囲と現在状態を可視化する。

## シナリオ4: 障害対応（運用ステージ）

```bash
teraflow incident create --title "DBが応答しない" --severity critical
teraflow incident list
teraflow incident close --id inc-001
```

運用ステージで障害を記録し、対応状況を追跡する。

## シナリオ5: 定期レビュー

```bash
teraflow dashboard show          # 全体進捗確認
teraflow schedule show           # スケジュール確認
teraflow changelog generate      # リリースノート生成
```

進捗・スケジュール・変更履歴をまとめてレビューし、報告資料を準備する。
