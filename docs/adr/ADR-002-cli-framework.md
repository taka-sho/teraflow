---
codd:
  node_id: "adr:002-cli-framework"
  title: "ADR-002 CLIフレームワーク選定"
  depends_on:
    - id: "adr:001-language"
      relation: derives_from
    - id: "req:cli-project-mgmt"
      relation: implements
---

# ADR-002: CLIフレームワーク選定

## ステータス

提案（Proposed）

## コンテキスト

ADR-001でGoを推奨言語とした前提で、CLIフレームワークを選定する。
teraflow CLIは以下の特性を持つ:

- **深いサブコマンド構造**: `teraflow stage advance`, `teraflow group member add` 等、最大3階層
- **フラグの複雑さ**: `--group`, `--stage`, `--format` 等のグローバル/ローカルフラグ
- **出力形式の柔軟性**: テーブル, JSON, YAML の切り替え
- **ヘルプ自動生成**: 50+のコマンドに対する一貫したヘルプ

## 候補比較

### cobra (spf13/cobra)

| 項目 | 評価 |
|------|------|
| **サブコマンド** | ◎ ネスト無制限。teraflowの3階層に最適 |
| **フラグ管理** | ◎ pflag統合。persistent/localフラグ。Viper連携で設定ファイル統合 |
| **ヘルプ生成** | ◎ 自動生成。カスタマイズ可能 |
| **補完** | ◎ Bash/Zsh/Fish/PowerShell 自動生成 |
| **採用実績** | ◎ kubectl, docker, gh, hugo, terraform |
| **コード生成** | ○ `cobra-cli` でボイラーplate生成 |

### urfave/cli (v2/v3)

| 項目 | 評価 |
|------|------|
| **サブコマンド** | ○ v3でネスト改善。ただしcobraほど柔軟ではない |
| **フラグ管理** | ○ 基本的なフラグ管理。設定ファイル統合は別途実装 |
| **ヘルプ生成** | ○ テンプレートベース |
| **補完** | △ 自動補完は限定的 |
| **採用実績** | ○ ghdl, nerdctl |
| **学習コスト** | ◎ cobraより学習コストが低い |

### kong (alecthomas/kong)

| 項目 | 評価 |
|------|------|
| **サブコマンド** | ○ 構造体タグベースで定義。宣言的 |
| **フラグ管理** | ○ 構造体フィールドで自動マッピング |
| **ヘルプ生成** | ○ 自動生成 |
| **補完** | △ プラグイン経由 |
| **採用実績** | △ cobraほどの採用実績はない |
| **型安全** | ◎ 構造体ベースで型安全 |

## 決定

**cobra を採用する。**

### 理由

1. **サブコマンド構造の親和性**: teraflowは50+コマンドを持つ深いサブコマンド構造。cobraのネスト構造が最適
2. **gh CLIとの一貫性**: GitHub CLIもcobraを使用。ghコマンドラップ（B-004）との統合時に知見を活かせる
3. **Viper連携**: `teraflow.yml` 等の設定ファイルをViper経由で統合管理可能
4. **エコシステム成熟度**: 10年以上の実績。ドキュメント・事例が豊富

### コマンド構造設計

```
cmd/
├── root.go              # teraflow ルートコマンド + グローバルフラグ
├── init.go              # teraflow init
├── stage/
│   ├── stage.go         # teraflow stage (親コマンド)
│   ├── current.go       # teraflow stage current
│   ├── advance.go       # teraflow stage advance
│   ├── activate.go      # teraflow stage activate
│   └── freeze.go        # teraflow stage freeze
├── phase/
│   ├── phase.go
│   ├── current.go
│   ├── advance.go
│   ├── skip.go
│   └── status.go
├── group/
│   ├── group.go
│   ├── propose.go
│   ├── define.go
│   ├── finalize.go
│   ├── rebalance.go
│   ├── status.go
│   ├── list.go
│   └── member/
│       ├── member.go
│       ├── add.go
│       ├── remove.go
│       ├── transfer.go
│       └── list.go
├── role/
│   ├── role.go
│   ├── list.go
│   ├── show.go
│   └── check.go
├── cycle/
├── schedule/
├── rework/
├── incident/
├── agent/
├── changelog/
├── dashboard/
├── harness/
└── setup/
    ├── setup.go         # teraflow setup (親コマンド)
    ├── labels.go        # teraflow setup labels
    ├── actions.go       # teraflow setup actions
    ├── templates.go     # teraflow setup templates
    └── codeowners.go    # teraflow setup codeowners
```

### グローバルフラグ

```go
// root.go
rootCmd.PersistentFlags().StringP("format", "f", "table", "Output format: table|json|yaml")
rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
rootCmd.PersistentFlags().String("config", "", "Config file path (default: .teraflow/teraflow.yml)")
```

### Viper統合

```
設定優先度（高→低）:
1. コマンドラインフラグ
2. 環境変数 (TERAFLOW_*)
3. .teraflow/teraflow.yml
4. デフォルト値
```

## 影響

- ディレクトリ構造設計（design:directory-structure）はこのコマンド構造に準拠する
- Phase1で実装するコマンドはcmd_081機能マトリクスのP1列に対応
- cobra + Viper の組み合わせで設定ファイル統合（sec21 teraflow.yml）を実現

## 代替案（ADR-001でTypeScript+Denoが採用された場合）

| フレームワーク | 備考 |
|--------------|------|
| Cliffy (Deno) | サブコマンド・補完対応。Denoエコシステムでは最有力 |
| Commander (Node) | 最も普及。ただしNode依存 |
| yargs (Node) | 宣言的定義。TypeScript対応良好 |
