---
codd:
  node_id: "docs:cmd-schedule"
  title: "schedule コマンドリファレンス"
  depends_on:
    - id: "design:cli-interface"
      relation: implements
---
# teraflow schedule

## 概要

マスタースケジュールの表示、完了予測、計画値設定を行うコマンド群。

## 使用方法

```bash
teraflow schedule <subcommand> [flags]
```

## サブコマンド

- `show`: マスタースケジュールを表示
- `predict`: 完了予測を算出
- `set`: スケジュール初期値を設定

## フラグ・オプション

- `teraflow schedule show`
  - `--group, -g <string>`: グループ指定（省略時は全体）
  - `--gantt`: テキストガント表示
- `teraflow schedule predict`
  - `--ai`: AI予測を利用
  - `--group, -g <string>`: グループ指定
- `teraflow schedule set`
  - `--group, -g <string>`: グループID（必須）
  - `--phase <string>`: フェーズ名（必須）
  - `--start <date>`: 開始予定日（YYYY-MM-DD, 必須）
  - `--end <date>`: 終了予定日（YYYY-MM-DD, 必須）
  - `--artifacts <int>`: 計画成果物数

## 出力例

```text
$ teraflow schedule predict --group group-auth
Group: group-auth
  Predicted completion: 2026-05-18
  Confidence: standard
```

## 使用例

```bash
teraflow schedule show --gantt
teraflow schedule set --group group-auth --phase requirements --start 2026-04-01 --end 2026-04-12 --artifacts 18
teraflow schedule predict --ai
```
