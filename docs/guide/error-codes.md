# Error Codes

teraflow CLI のエラーコード体系（`TF-CCNN` 形式）について説明します。

## エラーコード形式

`TF-CCNN`

- `TF`: teraflow プレフィックス
- `CC`: カテゴリコード（2文字）
- `NN`: カテゴリ内連番（01-99）

## カテゴリ一覧

| コード | カテゴリ | 説明 |
|--------|---------|------|
| CF | Config | 設定ファイル関連 |
| RB | RBAC | 権限・ロール関連 |
| GT | Gate | ゲート条件関連 |
| GH | GitHub | GitHub CLI/API関連 |
| AI | AI Provider | AI API関連 |
| IX | Index | インデックス関連 |
| CL | CLI Common | フラグ・引数関連 |
| AU | Audit | 監査ログ関連 |
| DC | Document | ドキュメント生成関連 |
| TR | Trace | 依存トレース関連 |
| SM | Summary | 要約/集約関連 |
| SK | Skill | スキル読込・選択関連 |
| HK | Hook | フック実行関連 |
| ST | State | プロジェクト状態関連 |
| IO | I/O | ファイル入出力関連 |

## exit code

| exit code | 意味 |
|-----------|------|
| 0 | 成功 |
| 1 | 一般エラー |
| 2 | Usage エラー（引数不正等） |
| 3 | 権限エラー（RBAC/Gate） |
| 4 | 外部依存エラー（gh/AI API） |
| 5 | 状態エラー（未初期化等） |

## エラーカタログの参照

```bash
teraflow doctor error TF-RB01   # 特定コードの詳細
teraflow doctor errors          # 全コード一覧
```

## カタログへの新規エラー追加方法

1. `docs/errors/err-tf-xx01.md` を CoDD frontmatter 付きで作成
2. `go generate ./internal/errors/...` を実行
3. `internal/errors/catalog_gen.go` が自動更新される
4. コードから `tferrors.New(tferrors.CodeXxxYyy)` で使用可能
