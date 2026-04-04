---
codd:
  node_id: "req:non-functional"
  title: "teraflow 非機能要件"
  depends_on:
    - id: "req:teraflow-overview"
      relation: derives_from
---

# teraflow 非機能要件

本ドキュメントはteraflow CLIの非機能要件を定義する。機能要件（docs/00〜05）を補完し、品質・保守性・運用性を保証するための基準を規定する。

---

## NF-001: テスト容易性

### 達成目標

テストが容易に実行でき、品質を継続的に検証できること。

### 要件

1. **ユニットテスト**: 全internal/パッケージに対して `go test ./...` で実行可能なテストを提供する
2. **E2Eテスト**: teraflowバイナリをビルドし、`init` → `stage` → `phase` → `rework` → `doctor` の一連フローを実行するE2Eテストを提供する
3. **カバレッジ**: コードカバレッジ70%以上を維持する
4. **テスト独立性**: 各テストは独立して実行可能。外部サービス（GitHub API, AI API）への依存はinterfaceモックで代替する
5. **テストデータ**: テスト用YAMLファイルは `t.TempDir()` で動的に生成する（testdata/の静的ファイルは補助的に使用）

### 測定方法

- `go test ./... -v` の全テストpass
- `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out` で70%以上
- CI（CI-001, CI-005）で自動チェック

---

## NF-002: コード品質

### 達成目標

一貫したコーディング規約と品質基準を自動で維持すること。

### 要件

1. **静的解析**: `golangci-lint` を採用し、以下のリンターを有効化する:
   - `govet` — Go標準の静的解析
   - `errcheck` — エラーの未チェック検出
   - `staticcheck` — 高度な静的解析
   - `gosimple` — コード簡素化の提案
   - `unused` — 未使用コードの検出
   - `goimports` — import文の整理
2. **設定ファイル**: `.golangci.yml` をリポジトリルートに配置し、チーム共通の設定を維持する
3. **CIゲート**: lint違反はPRマージをブロックする

### 測定方法

- `golangci-lint run ./...` がexit 0
- CI（CI-002）で自動チェック

---

## NF-003: ビルド再現性

### 達成目標

任意の環境で同一のビルド成果物を再現可能であること。

### 要件

1. **クロスプラットフォーム**: 以下のOS/アーキテクチャでビルド可能とする:
   - `linux/amd64`
   - `linux/arm64`
   - `darwin/amd64`
   - `darwin/arm64`
   - `windows/amd64`
2. **CGO無効**: `CGO_ENABLED=0` で静的リンクバイナリを生成する（外部Cライブラリ依存なし）
3. **バージョン埋め込み**: `main.version` を `ldflags` で注入する
   ```
   go build -ldflags "-s -w -X main.version=v1.0.0" -o teraflow .
   ```
4. **再現性**: `go.sum` をコミットし、依存バージョンを固定する
5. **CIでのビルド検証**: PR時にmatrix buildで全OS/archのビルド成功を確認する

### 測定方法

- 5プラットフォーム全てで `go build` 成功
- CI（CI-003）でmatrix build自動チェック

---

## NF-004: セキュリティ

### 達成目標

既知の脆弱性を含む依存関係を検出し、迅速に対応すること。

### 要件

1. **脆弱性スキャン**: `govulncheck` で依存関係の既知脆弱性を検出する
   ```
   go install golang.org/x/vuln/cmd/govulncheck@latest
   govulncheck ./...
   ```
2. **Dependabot**: GitHub Dependabotを設定し、以下を自動監視する:
   - `gomod` — Go依存関係（週次チェック）
   - `github-actions` — GitHub Actionsバージョン（週次チェック）
3. **シークレット管理**: API key等の機密情報はリポジトリにコミットしない。環境変数または`.env`（.gitignore対象）で管理する
4. **CIゲート**: govulncheck違反はPRマージをブロックする（Phase2で導入）

### 測定方法

- `govulncheck ./...` がexit 0
- Dependabot PRが週次で生成・レビューされている
- CI（CI-004）で自動チェック（Phase2）

---

## NF-005: 設計一貫性

### 達成目標

要件定義・ADR・設計書間の依存関係が整合していること。

### 要件

1. **codd scan**: `codd scan` でfrontmatterベースの依存グラフを構築し、以下を検証する:
   - 全ドキュメントにcodd frontmatterが付与されている
   - `depends_on` の参照先が存在する（ダングリングリファレンスなし）
   - 循環依存がない
2. **PRチェック**: PRで `docs/` 配下のファイルが変更された場合、`codd scan` を自動実行し整合性を検証する
3. **ノード命名規則**: `req:*`（要件）, `adr:*`（ADR）, `design:*`（設計書）の接頭辞を維持する

### 測定方法

- `codd scan` が WARNING/ERROR なしで完了
- CI（CI-006）で自動チェック（Phase2）

---

## NF-006: リリース容易性

### 達成目標

タグ付与のみで自動的にリリースビルド・配布が実行されること。

### 要件

1. **GoReleaser**: `.goreleaser.yaml` でリリースプロセスを定義する
2. **タグベースリリース**: `git tag v1.0.0 && git push --tags` でGitHub Actionsが自動起動し、以下を実行する:
   - 5プラットフォームのバイナリビルド
   - チェックサム生成
   - GitHub Releasesへのアップロード
   - CHANGELOGの自動生成
3. **バージョニング**: セマンティックバージョニング（SemVer）に従う
   - `v0.x.y` — Phase1開発中
   - `v1.0.0` — Phase1安定版リリース
4. **Homebrew**: 将来的にHomebrew tapでのインストールをサポートする（Phase2以降）

### 測定方法

- `goreleaser check` がexit 0
- タグpush → GitHub Releases に5バイナリが自動アップロードされる
- CD（CD-001）で自動実行

---

## NF-007: 保守性

### 達成目標

ドキュメントとコードの整合性を継続的に維持すること。

### 要件

1. **Markdownリンク検証**: `docs/` 配下のMarkdownファイル内リンク（内部リンク・外部リンク）の切れを検出する
2. **Markdownlint**: Markdownの一貫したスタイルを維持する（見出し階層、空行、コードブロック等）
3. **CHANGELOG**: `CHANGELOG.md` を継続的に更新する。GoReleaserのchangelog自動生成をベースに、手動で補足する
4. **依存関係の鮮度**: `go mod tidy` で不要な依存を除去。Dependabotで更新を追跡する

### 測定方法

- markdownlint がexit 0
- CI（QA-002）で自動チェック（Phase3）

---

## 要件マトリクス

| ID | カテゴリ | Phase1 | Phase2 | Phase3 |
|----|---------|--------|--------|--------|
| NF-001 | テスト容易性 | go test + E2E | +カバレッジCI | — |
| NF-002 | コード品質 | golangci-lint | — | — |
| NF-003 | ビルド再現性 | matrix build | — | — |
| NF-004 | セキュリティ | 手動govulncheck | +CI自動 +Dependabot | — |
| NF-005 | 設計一貫性 | 手動codd scan | +CI自動 | — |
| NF-006 | リリース容易性 | GoReleaser | +Homebrew | — |
| NF-007 | 保守性 | 手動lint | — | +CI自動 |
