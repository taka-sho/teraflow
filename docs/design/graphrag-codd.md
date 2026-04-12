---
codd:
  node_id: "design:graphrag-codd"
  title: "GraphRAG × CoDD 統合アーキテクチャ設計書"
  depends_on:
    - id: "docs:concepts"
      relation: extends
    - id: "design:internal-packages"
      relation: extends
  tags:
    - graphrag
    - codd
    - architecture
    - ai
---

# GraphRAG × CoDD 統合アーキテクチャ設計書

## 1. アーキテクチャ概要

### 1.1 目的

teraflow の CoDD 文書群に GraphRAG を導入し、以下を実現する:

- **要件トレーサビリティ強化**: 明示的 depends_on を超えた暗黙的関連の発見
- **影響分析の高度化**: 文書変更時の波及範囲を意味レベルで追跡
- **文脈検索**: 自然言語クエリによる関連文書の横断検索

### 1.2 責務分担

```text
┌─────────────────────────────────────────────────┐
│ teraflow CLI (Go)                               │
│                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │ cmd/     │  │ internal/│  │ internal/│      │
│  │ graph.go │  │ index/   │  │ trace/   │      │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘      │
│       │              │              │            │
│       │    CoDD 明示的グラフ        │            │
│       │    (index.yml, depends_on)  │            │
│       │              │              │            │
│       ▼              ▼              ▼            │
│  ┌─────────────────────────────────────────┐    │
│  │ internal/graphbridge/                    │    │
│  │  bridge.go  — subprocess 呼び出し       │    │
│  │  types.go   — Go↔Python 共有型定義      │    │
│  └──────────────────┬──────────────────────┘    │
└─────────────────────┼───────────────────────────┘
                      │ subprocess (stdin/stdout JSON)
                      ▼
┌─────────────────────────────────────────────────┐
│ teraflow-graphrag (Python)                      │
│                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │ extract/ │  │ graph/   │  │ query/   │      │
│  │ エンティ │  │ ナレッジ │  │ 検索     │      │
│  │ ティ抽出 │  │ グラフ   │  │ エンジン │      │
│  └──────────┘  └──────────┘  └──────────┘      │
│                                                 │
│  ストレージ: .teraflow/graphrag/                │
│  (JSON + NetworkX graphml)                      │
└─────────────────────────────────────────────────┘
```

### 1.3 Go CLI と Python モジュールの役割

| 責務 | Go CLI (teraflow) | Python (teraflow-graphrag) |
|------|-------------------|---------------------------|
| CoDD 明示的グラフ管理 | index.yml, depends_on, trace | - |
| エンティティ抽出 | - | LLM 呼び出し、テキスト解析 |
| ナレッジグラフ構築 | - | グラフ構築、コミュニティ検出 |
| グラフクエリ | サブプロセス経由で呼び出し | クエリ実行、結果返却 |
| ストレージ | .teraflow/index.yml | .teraflow/graphrag/ |
| CLI UX | コマンド体系、出力整形 | - |

### 1.4 通信方式: subprocess + JSON

**選定理由**:

| 方式 | 利点 | 欠点 | 評価 |
|------|------|------|------|
| **subprocess (stdin/stdout JSON)** | 依存ゼロ、デバッグ容易、サーバー不要 | 起動レイテンシ（~0.5s） | **採用** |
| gRPC | 型安全、高速、ストリーミング | protobuf 必須、サーバー常駐 | 過剰 |
| REST (FastAPI) | デバッグ容易 | HTTP オーバーヘッド、サーバー常駐 | 不適 |
| Embedded (cgo+CPython) | IPC なし | ビルド複雑、GIL | 不適 |

subprocess を選定。teraflow は CLI ツールであり、GraphRAG 操作は低頻度（インデックス構築: 手動/CI時、クエリ: 都度）。起動レイテンシ ~0.5s は許容範囲。サーバー常駐を回避し、単体 CLI のシンプルさを維持する。

**プロトコル**:

```json
// Go → Python (stdin)
{
  "command": "index" | "query" | "impact",
  "params": {
    "project_root": "/path/to/project",
    "changed_files": ["docs/requirements/req-auth.md"],
    "query": "認証に関連する設計文書"
  }
}

// Python → Go (stdout)
{
  "status": "ok" | "error",
  "result": { ... },
  "error": "エラーメッセージ（status=error時）"
}
```

## 2. グラフスキーマ設計

### 2.1 ノード型

CoDD の既存構造をベースに、GraphRAG が抽出するエンティティを追加:

```text
ノード型:
┌─────────────────────────────────────────────────────────┐
│ 明示的ノード（CoDD 由来、index.yml から取得）           │
│                                                         │
│  Document  — node_id, title, path, status, tags         │
│             CoDD 文書1つに対応                           │
└─────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────┐
│ 暗黙的ノード（GraphRAG 抽出）                           │
│                                                         │
│  Concept   — 技術概念、ドメイン用語                     │
│             例: "RBAC", "ゲート条件", "Leiden算法"       │
│                                                         │
│  Actor     — 役割、システムコンポーネント               │
│             例: "管理者", "teraflow CLI", "GitHub Actions"|
│                                                         │
│  Decision  — 設計決定、ADR                              │
│             例: "Go+cobra採用", "ラベルベース判定"       │
│                                                         │
│  Community — コミュニティ検出によるトピッククラスタ      │
│             例: "認証・権限系", "CI/CD系", "文書管理系"  │
└─────────────────────────────────────────────────────────┘
```

### 2.2 エッジ型

```text
エッジ型:
┌─────────────────────────────────────────────────────────┐
│ 明示的エッジ（CoDD 由来）                               │
│                                                         │
│  DEPENDS_ON     — Document → Document                   │
│                   source: codd frontmatter              │
│                   ※ 最優先。GraphRAG で重複抽出しない    │
└─────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────┐
│ 暗黙的エッジ（GraphRAG 抽出）                           │
│                                                         │
│  MENTIONS       — Document → Concept/Actor/Decision     │
│                   weight: 出現回数ベースの重み           │
│                                                         │
│  RELATED_TO     — Concept ↔ Concept                     │
│                   文書を跨いだ概念間の関連               │
│                                                         │
│  IMPLEMENTS     — Document → Decision                   │
│                   設計文書がADRを実装                     │
│                                                         │
│  MEMBER_OF      — Document/Concept → Community          │
│                   コミュニティ所属                       │
└─────────────────────────────────────────────────────────┘
```

### 2.3 スキーマ定義（Python 側 dataclass）

```python
from dataclasses import dataclass, field
from enum import Enum

class NodeType(Enum):
    DOCUMENT = "document"
    CONCEPT = "concept"
    ACTOR = "actor"
    DECISION = "decision"
    COMMUNITY = "community"

class EdgeType(Enum):
    DEPENDS_ON = "depends_on"       # CoDD 明示的
    MENTIONS = "mentions"           # GraphRAG 暗黙的
    RELATED_TO = "related_to"       # GraphRAG 暗黙的
    IMPLEMENTS = "implements"       # GraphRAG 暗黙的
    MEMBER_OF = "member_of"         # コミュニティ所属

@dataclass
class Node:
    id: str
    type: NodeType
    label: str
    properties: dict = field(default_factory=dict)
    source: str = "graphrag"  # "codd" | "graphrag"

@dataclass
class Edge:
    source_id: str
    target_id: str
    type: EdgeType
    weight: float = 1.0
    source: str = "graphrag"  # "codd" | "graphrag"
    properties: dict = field(default_factory=dict)
```

## 3. データフロー

### 3.1 インデックス構築フロー

```text
CoDD 文書作成/更新
    │
    ▼
[1] teraflow index build
    │  docs/**/*.md スキャン → index.yml 更新
    │  (既存: internal/index/builder.go)
    │
    ▼
[2] teraflow graph build [--full | --incremental]
    │
    ├─ Go側: index.yml 読み込み → 変更検出 (content_hash 比較)
    │         → 変更ファイルリスト + 全文書メタデータを JSON で Python へ
    │
    ├─ Python側:
    │   ├─ [2a] エンティティ抽出 (LLM)
    │   │       変更文書のみ処理 (incremental)
    │   │       → Concept, Actor, Decision ノード抽出
    │   │       → MENTIONS, RELATED_TO, IMPLEMENTS エッジ抽出
    │   │
    │   ├─ [2b] CoDD 明示的グラフ取り込み
    │   │       index.yml の depends_on → DEPENDS_ON エッジ
    │   │       ※ GraphRAG 抽出と重複しないようフィルタ
    │   │
    │   ├─ [2c] コミュニティ検出 (Leiden)
    │   │       → Community ノード + MEMBER_OF エッジ
    │   │
    │   └─ [2d] コミュニティ要約 (LLM)
    │           → 各コミュニティの概要テキスト生成
    │
    └─ 結果: .teraflow/graphrag/ に保存
        ├── graph.graphml          # NetworkX グラフ
        ├── entities.json          # 抽出エンティティ
        ├── communities.json       # コミュニティ情報
        ├── summaries/             # コミュニティ要約
        └── state.json             # 増分更新用メタデータ
```

### 3.2 クエリフロー

```text
[A] 影響分析クエリ
    teraflow graph impact <node_id>
    │
    ├─ Go側: node_id → Python subprocess
    ├─ Python側: DEPENDS_ON (明示) + MENTIONS/RELATED_TO (暗黙) を統合走査
    │            → 影響を受ける Document + 関連 Concept リストを返却
    └─ Go側: 結果整形して出力

[B] 文脈検索クエリ
    teraflow graph search "認証に関連する設計文書"
    │
    ├─ Go側: クエリ文字列 → Python subprocess
    ├─ Python側:
    │   ├─ Local Search: エンティティベースの関連文書検索
    │   └─ Global Search: コミュニティ要約ベースの広域検索
    └─ Go側: 結果整形して出力

[C] ステータス整合性クエリ
    teraflow graph check
    │
    ├─ Go側: index.yml (status 情報) → Python subprocess
    ├─ Python側: グラフ走査で矛盾検出
    │   例: status=draft の要件に DEPENDS_ON している status=confirmed の設計
    └─ Go側: 警告リストを出力
```

## 4. インクリメンタル更新戦略

### 4.1 課題

Microsoft GraphRAG は全再構築のみでインクリメンタル更新を未サポート（GitHub Discussion #511）。Leiden コミュニティ検出はグラフ全体の構造に依存するため、局所的な変更でもコミュニティ割り当てが全体的に変わりうる。

### 4.2 採用方式: ハイブリッド差分更新

```text
更新レイヤー別の戦略:

[L1] エンティティ抽出     → 差分更新可能 ✅
     変更文書のみ再抽出。旧エンティティを削除→新エンティティを追加。
     content_hash (index.yml) で変更検出。

[L2] 明示的エッジ更新     → 差分更新可能 ✅
     index.yml の depends_on 差分を直接反映。

[L3] 暗黙的エッジ更新     → 差分更新可能 ✅
     変更文書に関連するエッジのみ再計算。

[L4] コミュニティ検出     → 全再構築が必要 ⚠️
     Leiden は局所更新不可。ただし以下で緩和:
     - 閾値ベース再構築: エッジ変更率 > 10% の場合のみ再実行
     - それ以外: 前回のコミュニティ割り当てを維持
     - 強制再構築: --full フラグで明示的に全再構築

[L5] コミュニティ要約     → コミュニティ変更時のみ ✅
     L4 で変更されたコミュニティのみ再要約。
```

### 4.3 state.json による変更追跡

```json
{
  "version": "1",
  "last_build_at": "2026-04-09T22:00:00Z",
  "document_hashes": {
    "req-user-auth": "a1b2c3...",
    "design-api-gateway": "d4e5f6..."
  },
  "edge_change_count_since_last_community": 5,
  "community_rebuild_threshold": 0.10,
  "last_community_build_at": "2026-04-08T18:00:00Z"
}
```

### 4.4 LightRAG の採用検討

Microsoft GraphRAG の代替として **LightRAG** (HKUDS) を Phase 2 以降の実装候補とする:

| 比較項目 | Microsoft GraphRAG | LightRAG |
|----------|-------------------|----------|
| インクリメンタル更新 | 未サポート | **サポート済み** |
| クエリレイテンシ | 重い（map-reduce） | **~30% 軽量** |
| コミュニティ検出 | Leiden（全再構築） | デュアルレベル（軽量） |
| 依存関係 | 重い（graspologic等） | 軽量 |
| ベンチマーク | 基準 | **複数で上回る** |
| 成熟度 | Microsoft 公式 | EMNLP 2025 論文 |

**推奨**: Phase 2 実装時点で LightRAG の安定性を再評価し、採用を判断。設計上は抽象化層（GraphEngine インターフェース）を設け、バックエンド切り替え可能にする。

## 5. ストレージ選定

### 5.1 比較

| 方式 | 利点 | 欠点 | CLIツール適合性 |
|------|------|------|----------------|
| **NetworkX + GraphML** | Pythonネイティブ、ファイルベース、git管理可能 | 大規模時メモリ | **最適** |
| Neo4j | 高速クエリ、大規模対応 | サーバー必須、インストール負担 | 不適 |
| SQLite + JSON | 軽量、単一ファイル | グラフクエリが冗長 | 次善 |
| Parquet (GraphRAG デフォルト) | 列指向、分析向き | グラフ操作に不向き | 不適 |

### 5.2 推奨: NetworkX + GraphML + JSON補助ファイル

```text
.teraflow/graphrag/
├── graph.graphml          # メイングラフ（NetworkX GraphML形式）
│                            ノード属性: type, label, source, properties
│                            エッジ属性: type, weight, source
├── entities.json          # エンティティ詳細（抽出テキスト、出現位置等）
├── communities.json       # コミュニティ割り当て + メタデータ
├── summaries/
│   ├── community-0.txt    # コミュニティ要約テキスト
│   ├── community-1.txt
│   └── ...
└── state.json             # 増分更新用状態管理
```

**選定理由**:
- teraflow は CLI ツール。サーバー依存（Neo4j）は導入障壁が高すぎる
- GraphML は XML ベースでテキスト差分が取れる（git 管理可能）
- NetworkX はグラフ操作 API が豊富で、Leiden (graspologic) と直接連携
- 文書数 200 件規模なら NetworkX のインメモリ処理で十分（ノード数 ~1000, エッジ数 ~5000 想定）

## 6. Python モジュール構成

### 6.1 パッケージ構造

```text
graphrag/                          # monorepo 内の Python パッケージ
├── pyproject.toml                 # uv/pip 対応
├── teraflow_graphrag/
│   ├── __init__.py
│   ├── __main__.py                # CLI エントリポイント（subprocess 受信）
│   ├── config.py                  # 設定管理
│   │
│   ├── extract/
│   │   ├── __init__.py
│   │   ├── extractor.py           # LLM エンティティ抽出
│   │   └── prompts.py             # 抽出プロンプトテンプレート
│   │
│   ├── graph/
│   │   ├── __init__.py
│   │   ├── builder.py             # グラフ構築（NetworkX）
│   │   ├── community.py           # Leiden コミュニティ検出
│   │   ├── codd.py                # CoDD 明示的グラフ取り込み
│   │   └── schema.py              # ノード/エッジ型定義
│   │
│   ├── query/
│   │   ├── __init__.py
│   │   ├── engine.py              # クエリエンジン（GraphEngine インターフェース）
│   │   ├── impact.py              # 影響分析クエリ
│   │   ├── search.py              # 文脈検索クエリ
│   │   └── check.py               # ステータス整合性チェック
│   │
│   └── storage/
│       ├── __init__.py
│       ├── graphml.py              # GraphML 読み書き
│       └── state.py                # state.json 管理
│
└── tests/
    ├── test_extractor.py
    ├── test_builder.py
    ├── test_query.py
    └── fixtures/                   # テスト用CoDD文書
```

> **Note**: monorepo 構成の詳細は第 11 章を参照。

### 6.2 Go 側 bridge パッケージ

```go
// internal/graphbridge/bridge.go

package graphbridge

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
)

type Bridge struct {
    pythonCmd   string // "python3" or "uv run"
    modulePath  string // teraflow-graphrag パッケージパス
    projectRoot string
}

type Request struct {
    Command string         `json:"command"` // "index", "query", "impact", "check"
    Params  map[string]any `json:"params"`
}

type Response struct {
    Status string         `json:"status"` // "ok", "error"
    Result map[string]any `json:"result,omitempty"`
    Error  string         `json:"error,omitempty"`
}

func (b *Bridge) Execute(ctx context.Context, req Request) (*Response, error) {
    input, _ := json.Marshal(req)
    cmd := exec.CommandContext(ctx, b.pythonCmd, "-m", "teraflow_graphrag")
    cmd.Stdin = bytes.NewReader(input)
    cmd.Dir = b.projectRoot

    out, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("graphrag subprocess: %w", err)
    }

    var resp Response
    if err := json.Unmarshal(out, &resp); err != nil {
        return nil, fmt.Errorf("parse graphrag response: %w", err)
    }
    return &resp, nil
}
```

### 6.3 CLI コマンド設計

```text
teraflow graph build [--full] [--incremental] [--dry-run]
    インデックス構築。デフォルトは --incremental。

teraflow graph impact <node_id> [--depth N] [--format json]
    影響分析。明示的 + 暗黙的関係を統合して出力。

teraflow graph search "<query>" [--mode local|global] [--format json]
    文脈検索。local=エンティティベース、global=コミュニティ要約ベース。

teraflow graph check [--format json]
    ステータス整合性チェック。矛盾を警告として出力。

teraflow graph status [--format json]
    グラフ統計情報（ノード数、エッジ数、コミュニティ数、最終更新日時）。
```

## 7. コスト試算

### 7.1 前提

- エンティティ抽出: GPT-4o-mini ($0.15/1M input, $0.60/1M output)
- コミュニティ要約: Claude Haiku ($0.25/1M input, $1.25/1M output)
- チャンクサイズ: 600 トークン
- 平均文書サイズ: 3,000 トークン（5 チャンク/文書）
- エンティティ抽出 output: ~500 トークン/チャンク

### 7.2 規模別コスト概算

| 規模 | 文書数 | チャンク数 | 抽出コスト | 要約コスト | 合計 | 備考 |
|------|--------|-----------|-----------|-----------|------|------|
| **小** | 10 | 50 | $0.02 | $0.01 | **~$0.03** | 個人プロジェクト |
| **中** | 50 | 250 | $0.11 | $0.04 | **~$0.15** | チーム開発 |
| **大** | 200 | 1,000 | $0.45 | $0.15 | **~$0.60** | 大規模SI |

### 7.3 インクリメンタル更新時のコスト

差分更新では変更文書のみ再抽出。1 文書変更時:
- 抽出: 5 チャンク × (600+500) トークン ≈ $0.003
- コミュニティ再要約: 閾値以下なら $0（スキップ）

**日常的な更新コストはほぼ無視可能。**

### 7.4 モデル選定指針

| 用途 | 推奨モデル | 理由 |
|------|-----------|------|
| エンティティ抽出 | GPT-4o-mini / Claude Haiku | 低コスト、十分な精度 |
| コミュニティ要約 | Claude Haiku / GPT-4o-mini | 要約品質は中程度で十分 |
| クエリ応答 | GPT-4o / Claude Sonnet | 高精度が必要（ユーザー対面） |

teraflow の assignments 設定で `graphrag_extract` / `graphrag_summarize` / `graphrag_query` タイプを追加し、モデルルーティングを制御する。

## 8. 段階的導入計画

### Phase 1: CoDD 既存グラフの可視化・走査強化（Go 内完結）

**目的**: Python 依存なしで、既存の CoDD グラフの活用度を高める。

**実装内容**:
- `teraflow graph status`: index.yml の統計情報表示（ノード数、エッジ数、孤立ノード）
- `teraflow graph check`: depends_on の整合性チェック強化
  - 存在しない node_id への依存
  - status 矛盾（draft に依存する confirmed）
  - 循環依存検出
- `teraflow graph export --format dot|mermaid`: グラフ可視化出力
  - Mermaid: GitHub 上で直接表示可能
  - DOT: Graphviz でレンダリング

**価値**: GraphRAG なしでもトレーサビリティが向上。Phase 2 の基盤。

**見積り**: サブタスク 2-3 件、M 工数

### Phase 2: Python エンティティ抽出 + ナレッジグラフ構築

**目的**: CoDD 文書から暗黙的関係を抽出し、ナレッジグラフを構築する。

**実装内容**:
- teraflow-graphrag Python パッケージ作成
- `teraflow graph build`: エンティティ抽出 + グラフ構築
- internal/graphbridge/ Go ブリッジ
- .teraflow/graphrag/ ストレージ
- インクリメンタル更新（L1-L3）
- コミュニティ検出（Leiden、閾値ベース再構築）

**前提**: Python 3.11+、uv（パッケージ管理）

**価値**: 暗黙的関連の発見、概念横断検索の基盤。

**見積り**: サブタスク 5-6 件、L 工数

### Phase 3: GraphRAG クエリエンジン統合

**目的**: 自然言語クエリによる検索・影響分析を実現する。

**実装内容**:
- `teraflow graph search`: Local/Global 検索
- `teraflow graph impact`: 統合影響分析（明示的 + 暗黙的）
- コミュニティ要約生成 + Global Search
- query/ パッケージ実装
- LightRAG バックエンド検討（Phase 2 評価結果に基づく）

**価値**: 「この要件を変更したら何が影響を受けるか？」に回答可能。

**見積り**: サブタスク 4-5 件、L 工数

### 各 Phase の独立価値

```text
Phase 1 だけでも: グラフ可視化 + 整合性チェック → 文書管理の品質向上
Phase 1+2 で:     暗黙的関連発見 → 「見えていなかった依存」の検出
Phase 1+2+3 で:   自然言語検索 + 影響分析 → 完全なトレーサビリティ
```

## 9. CoDD 明示的関係 vs GraphRAG 暗黙的関係の矛盾解決ポリシー

### 9.1 基本原則: CoDD が常に優先

```text
優先順位:
  1. CoDD 明示的関係（depends_on）  — 人間が意図的に定義した依存
  2. GraphRAG 暗黙的関係            — AI が抽出した関連
```

### 9.2 重複抽出の防止

エンティティ抽出プロンプトに以下を含める:

```text
## 制約
以下の関係は CoDD frontmatter で既に定義済みです。
これらと同一の関係を抽出しないでください:
{既存の depends_on リスト}

上記以外の、文書内容から推定される暗黙的な関係のみを抽出してください。
```

### 9.3 矛盾検出と解決

| 矛盾パターン | 例 | 解決方針 |
|-------------|-----|---------|
| CoDD に依存あり、GraphRAG に関連なし | depends_on はあるが意味的関連が薄い | CoDD を維持。情報として「意味的関連が薄い可能性」を警告 |
| CoDD に依存なし、GraphRAG に強い関連 | depends_on 未定義だが内容が密接 | `graph check` で「depends_on 追加を推奨」として警告 |
| CoDD と GraphRAG で方向が逆 | A→B (CoDD) だが B→A (GraphRAG) | CoDD の方向を正とする。GraphRAG 側は双方向関連として保持 |

### 9.4 source フィールドによる区別

全ノード・エッジに `source` フィールド（"codd" | "graphrag"）を付与。クエリ結果で明示的/暗黙的の区別を表示:

```text
▲ Upstream (depends on):
  └─ [CoDD] req-session-management (セッション管理要件)
     └─ [CoDD] req-core-auth (コア認証基盤)

● req-user-auth (ユーザー認証要件)

▼ Related (GraphRAG):
  └─ [GraphRAG] design-api-gateway (APIゲートウェイ設計) — Concept: "認証トークン"
  └─ [GraphRAG] adr-003-ai-provider (AIプロバイダ選定) — Concept: "API認証"
```

## 10. リスクと代替案

### 10.1 主要リスク

| リスク | 影響 | 発生確率 | 緩和策 |
|--------|------|----------|--------|
| **LLM コスト超過** | 大規模プロジェクトで月額$10+ | 中 | インクリメンタル更新、低コストモデル選定、`--estimate-cost` フラグ |
| **Python 依存の導入障壁** | ユーザーに Python 環境を要求 | 高 | Phase 1 は Go 完結。Phase 2+ は `teraflow doctor` で環境チェック。uv による簡易インストール |
| **エンティティ抽出精度** | 不適切な関連の抽出 | 中 | source フィールドで明示的/暗黙的を区別。CoDD 優先ポリシー。`--dry-run` で事前確認 |
| **Leiden 全再構築コスト** | 文書数増加で再構築時間増大 | 低（200件以下） | 閾値ベース再構築。LightRAG 移行検討 |
| **保守性** | Go + Python 二重技術スタック | 中 | subprocess による疎結合。Python 側は独立パッケージとしてテスト・リリース可能 |

### 10.2 代替案

### 代替案 A: GraphRAG なし（CoDD グラフ強化のみ）
- Phase 1 のみ実装。既存の depends_on + tags による検索を強化。
- 利点: Python 不要、シンプル、低コスト
- 欠点: 暗黙的関連の発見不可、自然言語検索不可
- **判断**: Phase 1 を先行実装し、ユーザーフィードバックで Phase 2 以降の必要性を判断

### 代替案 B: 埋め込みベクトル検索（Embedding のみ）
- 文書を embedding して類似度検索。グラフ構築なし。
- 利点: シンプル、低コスト（embedding のみ）
- 欠点: グラフ構造なし、影響分析不可、コミュニティ検出なし
- **判断**: GraphRAG の方がCoDD のグラフ構造と親和性が高い。ただし Phase 2 実装時に比較検証の価値あり

### 代替案 C: Neo4j ベース
- Neo4j をグラフ DB として使用。neo4j-graphrag パッケージ活用。
- 利点: 高速クエリ、大規模対応、Cypher クエリ言語
- 欠点: サーバー常駐必須、インストール重い、CLI ツールに不適
- **判断**: エンタープライズ版（将来）の選択肢として残すが、CLI 版では不採用

## 11. monorepo ディレクトリレイアウト

### 11.1 決定事項

teraflow リポジトリを monorepo とし、Python GraphRAG パッケージを同リポジトリ内の `graphrag/` ディレクトリに配置する。別リポジトリにはしない。

### 11.2 レイアウト

```text
teraflow/
├── cmd/                           # Go CLI コマンド
│   ├── root.go
│   ├── graph.go                   # teraflow graph * コマンド群
│   └── ...
├── internal/
│   ├── graphbridge/               # Go → Python subprocess bridge
│   │   ├── bridge.go
│   │   └── types.go
│   ├── index/                     # CoDD index (既存)
│   ├── trace/                     # CoDD trace (既存)
│   ├── doc/                       # CoDD doc gen (既存)
│   └── ...
├── graphrag/                      # Python パッケージ (optional)
│   ├── pyproject.toml             # uv/pip 対応、Python 3.11+
│   ├── teraflow_graphrag/
│   │   ├── __init__.py
│   │   ├── __main__.py            # subprocess エントリポイント
│   │   ├── config.py
│   │   ├── extract/               # エンティティ抽出
│   │   ├── graph/                 # ナレッジグラフ構築
│   │   ├── query/                 # クエリエンジン
│   │   └── storage/               # GraphML + state 管理
│   └── tests/
│       ├── test_extractor.py
│       ├── test_builder.py
│       ├── test_query.py
│       └── fixtures/
├── docs/
│   └── design/
│       └── graphrag-codd.md       # 本設計書
├── .teraflow/
│   ├── index.yml                  # CoDD index (既存)
│   ├── summaries/                 # AI 要約キャッシュ (既存)
│   └── graphrag/                  # GraphRAG ストレージ (Phase 2+)
│       ├── graph.graphml
│       ├── entities.json
│       ├── communities.json
│       ├── summaries/
│       └── state.json
├── go.mod
├── go.sum
├── .goreleaser.yaml
└── .github/
    └── workflows/
        ├── ci.yml                 # Go テスト (既存)
        └── ci-graphrag.yml        # Python テスト (Phase 2+)
```

### 11.3 monorepo の利点

| 利点 | 説明 |
|------|------|
| 単一バージョン管理 | Go CLI と Python パッケージのバージョンを一致させやすい |
| 統合テスト容易 | E2E テスト（Go → Python subprocess）が同リポで完結 |
| CoDD 文書共有 | Python テストが `docs/` の CoDD 文書を直接参照可能 |
| CI 一元化 | paths filter で言語別に制御しつつ、単一リポで管理 |
| リリース連動 | GoReleaser と Python リリースを同タグで連動可能 |

### 11.4 Go と Python の分離ルール

- `go.mod` は `graphrag/` を無視（Go モジュールスコープ外）
- `graphrag/pyproject.toml` は Go コードを参照しない
- 唯一の接点は `internal/graphbridge/` の subprocess 呼び出し
- `.gitignore` に `graphrag/.venv/` を追加

## 12. optional 依存と graceful degradation

### 12.1 Bridge フォールバック設計

`internal/graphbridge/bridge.go` の検出ロジック:

```go
// Available checks if the GraphRAG Python module is usable.
func (b *Bridge) Available() (bool, string) {
    // Step 1: monorepo 内の graphrag/ ディレクトリ確認
    pyprojectPath := filepath.Join(b.projectRoot, "graphrag", "pyproject.toml")
    if _, err := os.Stat(pyprojectPath); err == nil {
        // Step 2: Python モジュールが import 可能か確認
        cmd := exec.Command("python3", "-m", "teraflow_graphrag", "--version")
        cmd.Dir = filepath.Join(b.projectRoot, "graphrag")
        if out, err := cmd.Output(); err == nil {
            return true, strings.TrimSpace(string(out))
        }
        return false, "graphrag/ found but module not installed. Run: cd graphrag && pip install -e ."
    }
    return false, "graphrag/ directory not found"
}
```

### 12.2 コマンド別フォールバック動作

| コマンド | Phase | graphrag 未検出時 |
|----------|-------|-------------------|
| `teraflow graph status` | Phase 1 | **正常動作** — index.yml ベースの統計のみ表示 |
| `teraflow graph check` | Phase 1 | **正常動作** — CoDD depends_on 整合性チェックのみ |
| `teraflow graph export` | Phase 1 | **正常動作** — CoDD グラフの可視化出力 |
| `teraflow graph build` | Phase 2 | **エラー** + インストール手順表示 |
| `teraflow graph search` | Phase 3 | **エラー** + インストール手順表示 |
| `teraflow graph impact` | Phase 3 | **部分動作** — CoDD 明示的関係のみで走査（暗黙的関係なし） |

### 12.3 エラーメッセージ

Phase 2/3 コマンドで graphrag 未検出時:

```text
Error [TF-GR01]: GraphRAG module not found.

teraflow graph build requires the GraphRAG Python module.

Install:
  cd graphrag && pip install -e .

Or with uv:
  cd graphrag && uv pip install -e .

Verify:
  teraflow doctor
```

### 12.4 impact コマンドの段階的応答

`teraflow graph impact` は graphrag の有無で応答が変わる:

```text
# graphrag あり（フル機能）
▲ Upstream (depends on):
  └─ [CoDD] req-session-management
▼ Related (GraphRAG):
  └─ [GraphRAG] design-api-gateway — Concept: "認証トークン"

# graphrag なし（CoDD のみ）
▲ Upstream (depends on):
  └─ [CoDD] req-session-management
ℹ GraphRAG not available — showing CoDD explicit relations only.
  Install graphrag for implicit relation analysis.
```

## 13. teraflow doctor 拡張

### 13.1 graphrag ステータス表示

`teraflow doctor` の出力に GraphRAG 環境チェックを追加:

**インストール済みの場合:**

```text
Checks:
  ✓ Config file (.github/teraflow.yml)
  ✓ Git repository
  ✓ Go version (1.25.8)
  ...
  ✓ Python 3.12.4
  ✓ teraflow-graphrag v0.1.0 (graphrag/)
  ✓ GraphRAG index (.teraflow/graphrag/graph.graphml)
    Nodes: 145, Edges: 387, Communities: 8
    Last build: 2026-04-09T22:00:00Z
```

**未インストールの場合:**

```text
Checks:
  ✓ Config file (.github/teraflow.yml)
  ✓ Git repository
  ✓ Go version (1.25.8)
  ...
  ✗ teraflow-graphrag not installed (optional)
    → GraphRAG features disabled (graph build/search)
    → Phase 1 features available (graph status/check/export)
    → Install: cd graphrag && pip install -e .
```

### 13.2 実装方針

```go
// cmd/doctor.go に checkGraphRAG() を追加

func checkGraphRAG(projectRoot string) CheckResult {
    bridge := graphbridge.NewBridge(projectRoot)
    available, msg := bridge.Available()
    if available {
        return CheckResult{
            Name:    "teraflow-graphrag",
            Status:  StatusOK,
            Message: fmt.Sprintf("teraflow-graphrag %s (graphrag/)", msg),
        }
    }
    return CheckResult{
        Name:    "teraflow-graphrag",
        Status:  StatusWarning, // Warning（optional のため Error ではない）
        Message: "teraflow-graphrag not installed (optional)",
        Hint:    "Install: cd graphrag && pip install -e .",
    }
}
```

## 14. CI 構成

### 14.1 既存 Go CI（変更なし）

`.github/workflows/ci.yml` — Go テスト、lint、build は従来通り。`graphrag/` の変更では発火しない。

### 14.2 Python CI（新規追加）

`.github/workflows/ci-graphrag.yml`:

```yaml
name: CI (GraphRAG)

on:
  push:
    paths:
      - 'graphrag/**'
    branches: [main]
  pull_request:
    paths:
      - 'graphrag/**'

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        python-version: ['3.11', '3.12', '3.13']
    steps:
      - uses: actions/checkout@v6

      - uses: actions/setup-python@v6
        with:
          python-version: ${{ matrix.python-version }}

      - name: Install dependencies
        working-directory: graphrag
        run: |
          pip install -e ".[dev]"

      - name: Lint
        working-directory: graphrag
        run: |
          ruff check .
          ruff format --check .

      - name: Test
        working-directory: graphrag
        run: |
          pytest tests/ -v --tb=short
```

### 14.3 統合テスト（E2E）

```yaml
  integration:
    runs-on: ubuntu-latest
    needs: [test]
    steps:
      - uses: actions/checkout@v6

      - uses: actions/setup-go@v6
        with:
          go-version-file: 'go.mod'

      - uses: actions/setup-python@v6
        with:
          python-version: '3.12'

      - name: Install teraflow
        run: go install ./...

      - name: Install graphrag
        working-directory: graphrag
        run: pip install -e .

      - name: E2E test
        run: |
          # Go → Python subprocess の統合テスト
          teraflow graph build --dry-run
          teraflow graph status
          teraflow doctor
```

### 14.4 paths filter まとめ

| ワークフロー | トリガーパス | 言語 |
|-------------|-------------|------|
| ci.yml | `cmd/**`, `internal/**`, `go.mod` 等 | Go |
| ci-graphrag.yml | `graphrag/**` | Python |
| ci-graphrag.yml (integration) | `graphrag/**` + `internal/graphbridge/**` | Go + Python |

## 15. インストール手順

### 15.1 ユースケース別インストール

| ユースケース | コマンド | GraphRAG 機能 |
|-------------|---------|---------------|
| teraflow 単体（Go CLI のみ） | `go install github.com/taka-sho/teraflow@latest` | Phase 1 のみ |
| graphrag 追加（pip） | `cd graphrag && pip install -e .` | Phase 1-3 全機能 |
| graphrag 追加（uv） | `cd graphrag && uv pip install -e .` | Phase 1-3 全機能 |
| フル導入 | 上記両方 | 全機能 |

### 15.2 開発環境セットアップ

```bash
# 1. リポジトリクローン
git clone https://github.com/taka-sho/teraflow.git
cd teraflow

# 2. Go CLI ビルド
go build -o teraflow .

# 3. GraphRAG セットアップ（optional）
cd graphrag
python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"

# 4. 動作確認
teraflow doctor
```

### 15.3 uv を使う場合

```bash
# uv がインストール済みの場合
cd graphrag
uv venv
uv pip install -e ".[dev]"
```

### 15.4 GoReleaser との連携

GoReleaser は Go バイナリのみをリリースする。Python パッケージは含まない。

```text
リリース成果物:
  - teraflow (Go バイナリ)        ← GoReleaser で自動ビルド
  - graphrag/ (Python パッケージ)  ← リポジトリ内にソースとして同梱
```

ユーザーは Go バイナリをインストール後、必要に応じて `graphrag/` を pip でインストールする。将来的に PyPI への公開も検討可能だが、初期は editable install のみ。
