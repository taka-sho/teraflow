---
codd:
  node_id: "docs:guide-graphrag"
  title: "GraphRAG 利用ガイド"
  depends_on:
    - id: "design:graphrag-codd"
      relation: implements
  tags:
    - graphrag
    - guide
---

# GraphRAG 利用ガイド

GraphRAG は CoDD ドキュメントを知識グラフ化し、検索・影響分析・整合性確認を補助する機能です。

## 1. インストール要件

- Python `3.11+`
- GraphRAG extras（`graphrag/` 配下でインストール）

```bash
cd graphrag
python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"
```

## 2. グラフ構築

まず CoDD index を最新化し、その後 GraphRAG を構築します。

```bash
teraflow index build
teraflow graph build
```

主要オプション:

- `--incremental`（デフォルト）: 差分更新
- `--full`: 全量再構築
- `--dry-run`: 実行内容の確認のみ

## 3. 検索

```bash
teraflow graph search "認証に関係する文書"
teraflow graph search "アーキテクチャ全体像" --mode global
```

- `--mode local`: ノード近傍検索（デフォルト）
- `--mode global`: コミュニティ要約ベース検索

## 4. 影響分析

```bash
teraflow graph impact design:auth --depth 2
```

- `--depth N`: 依存波及の探索深さ（デフォルト `2`）

## 5. 状態確認・検証・出力

```bash
teraflow graph status
teraflow graph check
teraflow graph export --format mermaid
teraflow graph export --format dot --output graph.dot
```

- `status`: ノード/エッジ統計を表示
- `check`: BrokenRef、循環依存、ステータス矛盾などを検査
- `export`: `mermaid` または `dot` 形式で可視化

## 6. 設定例

```yaml
# .teraflow/config.yaml
graphrag:
  model: claude-haiku-4-5-20251001
  max_tokens: 4096
  output_dir: .teraflow/graphrag
  community_threshold: 0.1
  incremental: true
```

現行実装では `.github/teraflow.yml` の `graphrag` セクションからも同等設定を読み込みます。
