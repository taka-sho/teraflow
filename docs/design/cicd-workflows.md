---
codd:
  node_id: "design:cicd-workflows"
  title: "CI/CDワークフロー設計"
  depends_on:
    - id: "req:non-functional"
      relation: implements
    - id: "req:teraflow-overview"
      relation: derives_from
---

# CI/CDワークフロー設計

## 1. ワークフロー一覧

| ID | ファイル | トリガー | 対応要件 | Phase | 概要 |
|----|---------|---------|---------|-------|------|
| CI-001 | `.github/workflows/test.yml` | PR, push main | NF-001 | Phase1 | ユニットテスト |
| CI-002 | `.github/workflows/lint.yml` | PR, push main | NF-002 | Phase1 | golangci-lint |
| CI-003 | `.github/workflows/build.yml` | PR, push main | NF-003 | Phase1 | クロスプラットフォームビルド |
| CD-001 | `.github/workflows/release.yml` | tag v* | NF-006 | Phase1 | GoReleaserリリース |
| CI-004 | `.github/workflows/security.yml` | PR, push main | NF-004 | Phase2 | govulncheck |
| CI-005 | `.github/workflows/coverage.yml` | PR, push main | NF-001 | Phase2 | カバレッジ計測 |
| CI-006 | `.github/workflows/codd-check.yml` | PR | NF-005 | Phase2 | codd scan整合性 |
| QA-003 | `.github/workflows/e2e.yml` | push main | NF-001 | Phase2 | E2Eテスト |
| QA-001 | `.github/dependabot.yml` | 週次（自動） | NF-004 | Phase3 | 依存関係自動更新 |
| QA-002 | `.github/workflows/docs-check.yml` | PR (docs/) | NF-007 | Phase3 | Markdownlint |

---

## 2. ワークフロー詳細設計

### CI-001: test.yml（ユニットテスト）

```yaml
name: Test

on:
  pull_request:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Run tests
        run: go test ./... -v -race

      - name: Check test count
        run: |
          count=$(go test ./... -v 2>&1 | grep -c "^--- PASS")
          echo "Tests passed: $count"
```

**設計判断:**
- `-race` フラグで競合状態を検出（データ競合はバグの温床）
- `ubuntu-latest` のみ（テスト自体はOS非依存。OS依存はCI-003で検証）
- Go 1.22固定（go.modのgoバージョンと合わせる）

---

### CI-002: lint.yml（コード品質）

```yaml
name: Lint

on:
  pull_request:
  push:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest
```

**設計判断:**
- `golangci-lint-action` はキャッシュ内蔵（高速）
- `.golangci.yml` で有効リンターを制御（NF-002で定義: govet, errcheck, staticcheck, gosimple, unused, goimports）

---

### CI-003: build.yml（クロスプラットフォームビルド）

```yaml
name: Build

on:
  pull_request:
  push:
    branches: [main]

jobs:
  build:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
          - os: ubuntu-latest
            goos: linux
            goarch: arm64
          - os: macos-latest
            goos: darwin
            goarch: amd64
          - os: macos-latest
            goos: darwin
            goarch: arm64
          - os: windows-latest
            goos: windows
            goarch: amd64

    runs-on: ${{ matrix.os }}

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Build
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: "0"
        run: go build -ldflags "-s -w -X main.version=ci-${{ github.sha }}" -o teraflow${{ matrix.goos == 'windows' && '.exe' || '' }} .

      - name: Verify binary
        if: matrix.goos != 'linux' || matrix.goarch == 'amd64'
        run: ./teraflow${{ matrix.goos == 'windows' && '.exe' || '' }} version
```

**設計判断:**
- `CGO_ENABLED=0` で静的リンク（NF-003）
- `ldflags` でバージョン注入（`main.version`）
- クロスコンパイル分（linux/arm64）はビルド成功のみ確認（実行はできない）
- windows/arm64は除外（NF-003の要件外）

---

### CD-001: release.yml（GoReleaserリリース）

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**設計判断:**
- `fetch-depth: 0` でCHANGELOG自動生成用の全履歴取得
- `permissions: contents: write` でRelease作成を許可
- GoReleaser v2系を使用

---

### CI-004: security.yml（脆弱性スキャン）— Phase2

```yaml
name: Security

on:
  pull_request:
  push:
    branches: [main]

jobs:
  govulncheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Install govulncheck
        run: go install golang.org/x/vuln/cmd/govulncheck@latest

      - name: Run govulncheck
        run: govulncheck ./...
```

---

### CI-005: coverage.yml（カバレッジ計測）— Phase2

```yaml
name: Coverage

on:
  pull_request:
  push:
    branches: [main]

jobs:
  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Run tests with coverage
        run: go test ./... -coverprofile=coverage.out

      - name: Check coverage threshold
        run: |
          total=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Total coverage: ${total}%"
          threshold=70
          if (( $(echo "$total < $threshold" | bc -l) )); then
            echo "::error::Coverage ${total}% is below threshold ${threshold}%"
            exit 1
          fi

      - name: Upload coverage
        if: github.event_name == 'push' && github.ref == 'refs/heads/main'
        uses: actions/upload-artifact@v4
        with:
          name: coverage-report
          path: coverage.out
```

---

### CI-006: codd-check.yml（codd scan整合性）— Phase2

```yaml
name: CoDD Check

on:
  pull_request:
    paths:
      - "docs/**"

jobs:
  codd-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-python@v5
        with:
          python-version: "3.11"

      - name: Install codd-dev
        run: pip install codd-dev

      - name: Run codd scan
        run: codd scan
```

**設計判断:**
- Python 3.11を使用（codd-dev v1.3.0の動作環境）
- `docs/**` パス変更時のみトリガー（不要な実行を防止）
- WARNINGは許容（frontmatter未付与のindex.md等）。ERRORのみ失敗扱い

---

### QA-003: e2e.yml（E2Eテスト）— Phase2

```yaml
name: E2E

on:
  push:
    branches: [main]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Build teraflow
        run: go build -o teraflow .

      - name: E2E test
        run: |
          # 初期化
          mkdir -p /tmp/e2e-project && cd /tmp/e2e-project
          git init && git config user.name "test" && git config user.email "test@test.com"

          # teraflow init
          $GITHUB_WORKSPACE/teraflow init --name "e2e-test" --non-interactive
          test -f .github/teraflow.yml || exit 1

          # teraflow doctor
          $GITHUB_WORKSPACE/teraflow doctor || true  # ghが未認証のためwarning許容

          # teraflow version
          $GITHUB_WORKSPACE/teraflow version | grep -q "v"
```

**設計判断:**
- main pushのみ（PR時はCI-001で十分）
- 実際のバイナリを使用したブラックボックステスト
- ghが未認証のためdoctorのgh項目はfailを許容

---

### QA-001: dependabot.yml — Phase3

```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
    labels:
      - "dependencies"
      - "go"
    open-pull-requests-limit: 5

  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
    labels:
      - "dependencies"
      - "github-actions"
    open-pull-requests-limit: 3
```

**設計判断:**
- 週次（月曜日）でチェック
- PR数上限設定（gomod: 5, actions: 3）で大量PRを防止
- ラベル自動付与でフィルタリング容易に

---

### QA-002: docs-check.yml（Markdownlint）— Phase3

```yaml
name: Docs Check

on:
  pull_request:
    paths:
      - "docs/**"

jobs:
  markdownlint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: DavidAnson/markdownlint-cli2-action@v16
        with:
          globs: "docs/**/*.md"
```

---

## 3. .goreleaser.yaml 設計

```yaml
# .goreleaser.yaml
version: 2

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - "-s -w -X main.version={{.Version}}"

archives:
  - format: tar.gz
    name_template: "teraflow_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^chore:"
      - "^ci:"
```

**設計判断:**
- `windows/arm64` を除外（需要が少ない）
- `-s -w` でデバッグ情報を除去（バイナリサイズ削減）
- `{{.Version}}` はGoReleaserがタグから自動注入
- CHANGELOGからdocs/chore/ciコミットを除外

---

## 4. 実装分担案

| 担当 | ワークフロー | 成果物 |
|------|------------|--------|
| **足軽1号** | Phase1: CI-001, CI-002, CI-003, CD-001 | test.yml, lint.yml, build.yml, release.yml, .goreleaser.yaml, .golangci.yml |
| **足軽2号** | Phase2: CI-004, CI-005, CI-006, QA-003 | security.yml, coverage.yml, codd-check.yml, e2e.yml |
| **足軽3号** | Phase3: QA-001, QA-002 | dependabot.yml, docs-check.yml |
| **足軽4号** | commit + push | 全員完了後に一括コミット |

### 実装順序

```
足軽1号（Phase1）→ 足軽2号（Phase2）→ 足軽3号（Phase3）→ 足軽4号（commit）
                                                                    ↓
                                                            codd scan で確認
```

Phase1のCI/CDが最優先。Phase2以降は並列実装可能。

### 足軽への注意事項

1. **CD-001担当（足軽1号）**: `.goreleaser.yaml` はリポジトリルートに配置。`goreleaser check` で構文確認してからコミット
2. **CI-006担当（足軽2号）**: `actions/setup-python@v5` + `python-version: "3.11"` が必要。codd-devはPythonパッケージ
3. **QA-001担当（足軽3号）**: `.github/dependabot.yml` はworkflowsディレクトリではなく `.github/` 直下に配置
4. **全員共通**: `.github/` ディレクトリは既に存在する（teraflow.yml, project-state.yml がある）
5. **main.goのversion変数**: `var version = "v0.1.0"` 形式で定義済み。ldflagsで上書き可能
