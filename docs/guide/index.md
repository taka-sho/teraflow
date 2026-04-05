---
codd:
  node_id: "docs:guide-index-summary"
  title: "Index・Summary キャッシュ利用ガイド"
  depends_on:
    - id: "design:ai-conversation-platform"
      relation: implements
---

# Index・Summary キャッシュ利用ガイド

## 1. 概要

`teraflow` は `docs/` 配下の CoDD 文書を走査し、`.teraflow/index.yml` と `.teraflow/summaries/` を使って AI への入力コンテキストを制御する。

- `index.yml`: 文書メタ情報（node_id, path, hash, summary有無）
- `summaries/`: 文書ごとの要約キャッシュ（再利用可能）

この仕組みにより、長文ドキュメント群でもトークン上限を守りながら関連文書を提示できる。

## 2. 前提条件（CoDD frontmatter）

`index build` で文書が取り込まれるには、対象Markdown先頭に frontmatter が必要。

```yaml
---
codd:
  node_id: "req:example"
  title: "Example Requirement"
  depends_on:
    - id: "design:base"
  tags:
    - requirements
---
```

最低条件:

- `codd.node_id` が空でない
- ファイルが `docs/**/*.md` 配下

## 3. `teraflow index build`

```bash
teraflow index build
```

実行内容:

- `docs/` 配下の markdown を走査
- frontmatter 付き文書を抽出
- `.teraflow/index.yml` を生成/更新

## 4. `teraflow index status`

```bash
teraflow index status
```

確認できる内容:

- indexエントリ数
- 生成時刻
- summary coverage（要約あり/全体）

## 5. `teraflow summary update`

```bash
teraflow summary update
```

実行内容:

- `index.yml` を読み込み
- 未生成または無効化された要約のみ再生成
- 結果を `.teraflow/summaries/*.txt` に保存

主要オプション:

- `--dry-run`: 更新対象のみ確認
- `--force`: 全要約を再生成

## 6. `teraflow summary show <node_id>`

```bash
teraflow summary show req:example
```

指定 `node_id` のキャッシュ要約を表示する。要約内容の確認やプロンプト調整時の点検に使う。

## 7. トークン配分の考え方

コンテキスト構築時は概ね次の枠で配分される。

- system prompt
- 会話履歴
- 関連文書要約（複数）
- 関連文書全文（最重要1件）
- 最新ユーザー入力

`max_context_tokens` を超える場合は、要約/全文を切り詰めて上限内に収める。
