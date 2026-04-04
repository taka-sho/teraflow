---
codd:
  node_id: "adr:001-language"
  title: "ADR-001 実装言語の選定"
  depends_on:
    - id: "req:teraflow-overview"
      relation: implements
    - id: "req:cli-project-mgmt"
      relation: implements
---

# ADR-001: 実装言語の選定

## ステータス

提案（Proposed）

## コンテキスト

teraflow CLIは以下の要件を満たす実装言語を選定する必要がある:

- **B-001裁定**: 言語は任意。実行時スピード重視
- **配布容易性**: エンジニアが即座にインストールできるシングルバイナリが望ましい
- **GitHub連携**: `gh` コマンドラップ方式（B-004）でGitHub操作を行う
- **YAML/Markdown処理**: project-state.yml, groups.yml, master-schedule.yml 等のYAMLファイル読み書き、CoDD frontmatter（Markdown YAML frontmatter）の解析
- **Phase1スコープ**: ローカルCLI（エンジニア向け）。Phase2以降でGitHub Actions連携

## 候補比較

### Go

| 項目 | 評価 |
|------|------|
| **実行速度** | ◎ コンパイル言語。GCありだがCLI用途では問題にならない |
| **シングルバイナリ** | ◎ `go build` で単一バイナリ。クロスコンパイル容易 |
| **CLIエコシステム** | ◎ cobra, urfave/cli 等の成熟フレームワーク |
| **YAML/Markdown** | ○ go-yaml, goldmark 等。十分だが型定義が冗長 |
| **GitHub CLI連携** | ◎ gh自体がGoで実装。go-gh ライブラリで直接統合も可能 |
| **AI SDK** | ○ Anthropic/OpenAI公式SDK対応 |
| **開発速度** | △ 型定義・エラーハンドリングの記述量が多い |

### TypeScript + Deno

| 項目 | 評価 |
|------|------|
| **実行速度** | ○ V8エンジン。起動時間はGoに劣るがCLI用途では許容範囲 |
| **シングルバイナリ** | ○ `deno compile` で単一バイナリ生成可能（サイズはGoより大きい） |
| **CLIエコシステム** | ○ Cliffy(Deno), Commander(Node)。成熟度はGoに劣る |
| **YAML/Markdown** | ◎ yaml, remark/unified エコシステム。frontmatter処理が容易 |
| **GitHub CLI連携** | ○ child_process経由でghコマンド実行 |
| **AI SDK** | ◎ Anthropic/OpenAI公式TypeScript SDKが最も充実 |
| **開発速度** | ◎ 動的型付け+型推論。プロトタイピングが速い |

### Rust

| 項目 | 評価 |
|------|------|
| **実行速度** | ◎ 最高レベルの実行性能。GCなし |
| **シングルバイナリ** | ◎ 静的リンクで最小バイナリ |
| **CLIエコシステム** | ◎ clap が非常に成熟 |
| **YAML/Markdown** | ○ serde_yaml, pulldown-cmark。機能十分 |
| **GitHub CLI連携** | ○ Command経由でghコマンド実行 |
| **AI SDK** | △ 公式SDKなし。コミュニティSDKのみ |
| **開発速度** | × 学習コスト高。所有権システムで記述量が多い |

## 総合評価

| 言語 | 実行速度 | 配布 | エコシステム | AI連携 | 開発速度 | 総合 |
|------|---------|------|------------|--------|---------|------|
| **Go** | ◎ | ◎ | ◎ | ○ | △ | **A** |
| **TypeScript+Deno** | ○ | ○ | ○ | ◎ | ◎ | **A** |
| Rust | ◎ | ◎ | ◎ | △ | × | B |

## 決定

**Go を推奨（第一候補）。TypeScript+Deno を代替候補とする。**

### 推奨理由

1. **B-001（実行時スピード重視）との整合**: Goはコンパイル言語で実行速度に優れる。CLIの起動時間も極めて短い
2. **シングルバイナリ配布**: `go build` 一発でクロスプラットフォームバイナリ生成。`brew`, `go install`, GitHub Releases で配布容易
3. **GitHub CLIとの親和性**: `gh` 自体がGoで書かれており、`go-gh` ライブラリでプログラマティックな統合も将来的に可能（B-004の発展形）
4. **CLIフレームワークの成熟度**: cobraは業界標準（kubectl, docker CLI, gh が採用）

### TypeScript+Denoが有利なケース

- AI SDK連携が中心的な機能になる場合（Phase2以降のAgent Pipeline）
- フロントエンド（ダッシュボード等）との技術スタック統一が重要な場合
- プロトタイピング速度を最優先する場合

### 配布方式

| 方式 | 対象 |
|------|------|
| `go install github.com/taka-sho/teraflow@latest` | Go開発者 |
| `brew install teraflow` (Homebrew tap) | macOS/Linux |
| GitHub Releases (バイナリ) | 全プラットフォーム |

## 影響

- ADR-002（CLIフレームワーク選定）はGo前提で cobra を第一候補とする
- ADR-003（AI連携）はGo用SDKの成熟度を考慮した設計が必要
- ADR-004（データ/GitHub連携）は go-yaml, go-gh を前提とする

## 備考

- 殿の最終裁定により言語が変更される場合、ADR-002〜004も連動して更新が必要
- Phase4（SaaS化）の際にはバックエンドAPI言語として再評価の余地あり
