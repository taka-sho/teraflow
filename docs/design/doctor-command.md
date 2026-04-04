---
codd:
  node_id: "design:doctor-command"
  title: "doctorコマンド詳細設計"
  depends_on:
    - id: "req:cli-project-mgmt"
      relation: implements
    - id: "design:internal-packages"
      relation: uses
    - id: "design:cli-interface"
      relation: extends
---

# doctorコマンド詳細設計

## 1. コマンドインターフェース

```
teraflow doctor [flags]

Flags:
  --check-ai        AI連携チェックも実行（デフォルト: skip）
  --format <type>   出力形式: text | json（デフォルト: text）
  --verbose, -v     詳細出力（チェック過程を表示）

Exit codes:
  0   全チェック pass
  1   1件以上の ✗ あり
```

## 2. チェック項目設計（4カテゴリ）

### Category 1: Environment

外部ツールのインストール・認証状態を確認する。

| チェック名 | 実装 | Pass条件 | メッセージ例 |
|-----------|------|---------|------------|
| `go` | `exec.LookPath("go")` + `exec.Command("go", "version")` | LookPath成功 + version出力取得 | `✓ go: go1.22.0` / `✗ go: not found` |
| `gh` | `exec.LookPath("gh")` + `exec.Command("gh", "auth", "status")` | LookPath成功 + auth status exit 0 | `✓ gh: authenticated as user-a` / `✗ gh: not authenticated` |
| `git` | `exec.LookPath("git")` | LookPath成功 | `✓ git: found` / `✗ git: not found` |

#### 実装

```go
func checkEnvironment() []CheckResult {
    var results []CheckResult

    // go
    if goPath, err := exec.LookPath("go"); err != nil {
        results = append(results, CheckResult{
            Category: "environment", Name: "go", OK: false,
            Message: "not found in PATH",
        })
    } else {
        out, _ := exec.Command(goPath, "version").Output()
        ver := strings.TrimSpace(string(out))
        results = append(results, CheckResult{
            Category: "environment", Name: "go", OK: true,
            Message: ver,
        })
    }

    // gh
    if ghPath, err := exec.LookPath("gh"); err != nil {
        results = append(results, CheckResult{
            Category: "environment", Name: "gh", OK: false,
            Message: "not found in PATH",
        })
    } else {
        cmd := exec.Command(ghPath, "auth", "status")
        out, err := cmd.CombinedOutput()
        if err != nil {
            results = append(results, CheckResult{
                Category: "environment", Name: "gh", OK: false,
                Message: "not authenticated",
            })
        } else {
            // 出力から "Logged in to github.com account XXX" を抽出
            msg := extractGHUser(string(out))
            results = append(results, CheckResult{
                Category: "environment", Name: "gh", OK: true,
                Message: msg,
            })
        }
    }

    // git
    if _, err := exec.LookPath("git"); err != nil {
        results = append(results, CheckResult{
            Category: "environment", Name: "git", OK: false,
            Message: "not found in PATH",
        })
    } else {
        results = append(results, CheckResult{
            Category: "environment", Name: "git", OK: true,
            Message: "found",
        })
    }

    return results
}
```

### Category 2: Configuration

teraflow設定ファイルの存在・構文・必須フィールドを検証する。

| チェック名 | 実装 | Pass条件 | メッセージ例 |
|-----------|------|---------|------------|
| `teraflow.yml` | `os.Stat` + `config.Load` | ファイル存在 + YAML parse成功 + version非空 | `✓ teraflow.yml: valid (project: my-project)` / `✗ teraflow.yml: not found` |
| `project-state.yml` | `os.Stat` + `state.LoadState` | ファイル存在 + YAML parse成功 + current_stage非空 | `✓ project-state.yml: valid (stage: initial_development)` / `✗ project-state.yml: parse error: ...` |

#### 実装

```go
func checkConfiguration(configPath string) []CheckResult {
    var results []CheckResult

    // teraflow.yml
    cfg, err := config.Load(configPath)
    if err != nil {
        results = append(results, CheckResult{
            Category: "configuration", Name: "teraflow.yml", OK: false,
            Message: err.Error(),
        })
    } else if cfg.Version == "" {
        results = append(results, CheckResult{
            Category: "configuration", Name: "teraflow.yml", OK: false,
            Message: "missing required field: version",
        })
    } else {
        results = append(results, CheckResult{
            Category: "configuration", Name: "teraflow.yml", OK: true,
            Message: fmt.Sprintf("valid (project: %s)", cfg.Project.Name),
        })
    }

    // project-state.yml
    s, err := state.LoadState(configPath)
    if err != nil {
        results = append(results, CheckResult{
            Category: "configuration", Name: "project-state.yml", OK: false,
            Message: err.Error(),
        })
    } else if s.Lifecycle.CurrentStage == "" {
        results = append(results, CheckResult{
            Category: "configuration", Name: "project-state.yml", OK: false,
            Message: "missing required field: lifecycle.current_stage",
        })
    } else {
        results = append(results, CheckResult{
            Category: "configuration", Name: "project-state.yml", OK: true,
            Message: fmt.Sprintf("valid (stage: %s)", s.Lifecycle.CurrentStage),
        })
    }

    return results
}
```

### Category 3: Project Integrity

運用データファイルの構文チェック。ファイル不在はOK（まだ手戻り/障害が発生していない正常状態）。

| チェック名 | 実装 | Pass条件 | メッセージ例 |
|-----------|------|---------|------------|
| `rework-log.yml` | `state.LoadReworkLog` | ファイル不在=OK、存在+parse成功=OK | `✓ rework-log.yml: 3 entries` / `✓ rework-log.yml: not yet created (ok)` / `✗ rework-log.yml: parse error` |
| `incident-log.yml` | `state.LoadIncidentLog` | 同上 | 同上 |

#### 実装

```go
func checkProjectIntegrity(configPath string) []CheckResult {
    var results []CheckResult

    // rework-log.yml
    rlog, err := state.LoadReworkLog(configPath)
    if err != nil {
        results = append(results, CheckResult{
            Category: "integrity", Name: "rework-log.yml", OK: false,
            Message: fmt.Sprintf("parse error: %v", err),
        })
    } else if len(rlog.Reworks) == 0 {
        results = append(results, CheckResult{
            Category: "integrity", Name: "rework-log.yml", OK: true,
            Message: "not yet created (ok)",
        })
    } else {
        results = append(results, CheckResult{
            Category: "integrity", Name: "rework-log.yml", OK: true,
            Message: fmt.Sprintf("%d entries", len(rlog.Reworks)),
        })
    }

    // incident-log.yml（同様のパターン）
    // state.LoadIncidentLog は state パッケージに追加が必要
    // LoadReworkLog と同じ構造（ファイル不在=空ログ返却）

    return results
}
```

### Category 4: AI Integration（`--check-ai` フラグ時のみ）

| チェック名 | 実装 | Pass条件 | メッセージ例 |
|-----------|------|---------|------------|
| `ANTHROPIC_API_KEY` | `os.Getenv("ANTHROPIC_API_KEY")` | 非空 | `✓ ANTHROPIC_API_KEY: set` / `✗ ANTHROPIC_API_KEY: not set` |
| `OPENAI_API_KEY` | `os.Getenv("OPENAI_API_KEY")` | 非空（defaultProviderがopenaiの場合のみ） | 同上 |

#### 実装

```go
func checkAIIntegration(configPath string) []CheckResult {
    var results []CheckResult

    // デフォルトプロバイダの確認
    cfg, err := config.Load(configPath)
    provider := "anthropic"
    if err == nil && cfg.AI.DefaultProvider != "" {
        provider = cfg.AI.DefaultProvider
    }

    // プロバイダに応じた環境変数チェック
    envKeys := map[string]string{
        "anthropic": "ANTHROPIC_API_KEY",
        "openai":    "OPENAI_API_KEY",
        "gemini":    "GOOGLE_API_KEY",
    }

    if envKey, ok := envKeys[provider]; ok {
        val := os.Getenv(envKey)
        if val == "" {
            results = append(results, CheckResult{
                Category: "ai", Name: envKey, OK: false,
                Message: "not set",
            })
        } else {
            results = append(results, CheckResult{
                Category: "ai", Name: envKey, OK: true,
                Message: "set",
            })
        }
    }

    return results
}
```

## 3. CheckResult 型

```go
// CheckResult は1つのチェック項目の結果を表す。
// cmd/doctor.go 内に定義する（internal/ に切り出すほどの複雑さはない）。
type CheckResult struct {
    Category string `json:"category"`
    Name     string `json:"name"`
    OK       bool   `json:"ok"`
    Message  string `json:"message"`
}
```

## 4. メイン関数

```go
// runChecks は全チェックを実行し、結果のスライスを返す。
// configPath: --config フラグの値
// checkAI: --check-ai フラグの値
func runChecks(configPath string, checkAI bool) []CheckResult {
    var all []CheckResult

    all = append(all, checkEnvironment()...)
    all = append(all, checkConfiguration(configPath)...)
    all = append(all, checkProjectIntegrity(configPath)...)

    if checkAI {
        all = append(all, checkAIIntegration(configPath)...)
    }

    return all
}

// hasFailure は結果に1件以上の ✗ があるかを返す。
func hasFailure(results []CheckResult) bool {
    for _, r := range results {
        if !r.OK {
            return true
        }
    }
    return false
}
```

## 5. 出力形式

### テキスト出力（デフォルト）

```
$ teraflow doctor

  Environment
    ✓ go: go1.22.0 linux/amd64
    ✓ gh: authenticated as user-a
    ✓ git: found

  Configuration
    ✓ teraflow.yml: valid (project: my-order-system)
    ✓ project-state.yml: valid (stage: initial_development)

  Project Integrity
    ✓ rework-log.yml: not yet created (ok)
    ✓ incident-log.yml: not yet created (ok)

  All checks passed.
```

問題ありの場合:

```
$ teraflow doctor

  Environment
    ✓ go: go1.22.0 linux/amd64
    ✗ gh: not authenticated
    ✓ git: found

  Configuration
    ✓ teraflow.yml: valid (project: my-order-system)
    ✗ project-state.yml: missing required field: lifecycle.current_stage

  Project Integrity
    ✓ rework-log.yml: 2 entries
    ✗ incident-log.yml: parse error: yaml: line 5: mapping values are not allowed

  3 issues found. Fix the above problems.
  Exit code: 1
```

### JSON出力（--format json）

```json
{
  "results": [
    {"category": "environment", "name": "go", "ok": true, "message": "go1.22.0 linux/amd64"},
    {"category": "environment", "name": "gh", "ok": false, "message": "not authenticated"},
    {"category": "environment", "name": "git", "ok": true, "message": "found"},
    {"category": "configuration", "name": "teraflow.yml", "ok": true, "message": "valid (project: my-order-system)"},
    {"category": "configuration", "name": "project-state.yml", "ok": true, "message": "valid (stage: initial_development)"},
    {"category": "integrity", "name": "rework-log.yml", "ok": true, "message": "not yet created (ok)"},
    {"category": "integrity", "name": "incident-log.yml", "ok": true, "message": "not yet created (ok)"}
  ],
  "pass": false,
  "issues": 1
}
```

## 6. cobra コマンド登録

```go
// cmd/doctor.go

func newDoctorCmd() *cobra.Command {
    var checkAI bool

    cmd := &cobra.Command{
        Use:   "doctor",
        Short: "Check project health and environment",
        RunE: func(cmd *cobra.Command, args []string) error {
            configPath, _ := cmd.Flags().GetString("config")
            format, _ := cmd.Flags().GetString("format")

            results := runChecks(configPath, checkAI)

            if format == "json" {
                return printJSON(results)
            }
            printText(results)

            if hasFailure(results) {
                os.Exit(1)
            }
            return nil
        },
    }

    cmd.Flags().BoolVar(&checkAI, "check-ai", false, "Check AI integration (API keys)")
    return cmd
}
```

`cmd/root.go` に登録:

```go
// root.go の newRootCmd 内
rootCmd.AddCommand(newInitCmd())
rootCmd.AddCommand(newDoctorCmd())  // 追加
```

## 7. テスト方針

```go
// cmd/doctor_test.go

func TestRunChecks_AllPass(t *testing.T) {
    // tmpdir に .github/teraflow.yml と .github/project-state.yml を配置
    // runChecks を直接呼び出し
    // 全結果の OK == true を検証
}

func TestRunChecks_MissingConfig(t *testing.T) {
    // tmpdir に teraflow.yml を配置しない
    // configuration カテゴリの teraflow.yml が OK == false であることを検証
}

func TestRunChecks_InvalidYAML(t *testing.T) {
    // tmpdir に不正なYAMLの teraflow.yml を配置
    // configuration カテゴリの teraflow.yml が OK == false であることを検証
}

func TestRunChecks_ReworkLogNotExist(t *testing.T) {
    // .teraflow/rework-log.yml が存在しない場合
    // integrity カテゴリの rework-log.yml が OK == true（not yet created）であることを検証
}

func TestRunChecks_AICheckSkipped(t *testing.T) {
    // checkAI=false の場合、ai カテゴリの結果が含まれないことを検証
}

func TestRunChecks_AICheckEnabled(t *testing.T) {
    // checkAI=true + 環境変数設定
    // ai カテゴリの結果が含まれ、OK == true であることを検証
}

func TestHasFailure(t *testing.T) {
    // 全 OK → false
    // 1件 NG → true
}

// 環境チェックのテスト:
// go/gh/git の LookPath テストは PATH 環境変数を操作してモック可能
func TestCheckEnvironment_MissingGo(t *testing.T) {
    t.Setenv("PATH", "/nonexistent")
    results := checkEnvironment()
    // go, gh, git 全て OK == false
}
```

## 8. internal/state への追加が必要な型・関数

doctorコマンドの実装にあたり、`internal/state` に以下の追加が必要:

```go
// IncidentEntry は障害記録1件。
type IncidentEntry struct {
    ID        string    `yaml:"id"`
    Title     string    `yaml:"title"`
    Severity  string    `yaml:"severity"`
    CreatedAt time.Time `yaml:"created_at"`
    Status    string    `yaml:"status"`
}

// IncidentLog は .teraflow/incident-log.yml のルート構造。
type IncidentLog struct {
    Incidents []IncidentEntry `yaml:"incidents"`
}

// LoadIncidentLog は .teraflow/incident-log.yml をロードする。
// ファイル不在時は空の IncidentLog を返す。
func LoadIncidentLog(configPath string) (*IncidentLog, error)
```

パスは `.teraflow/incident-log.yml`（rework-log.yml と同じディレクトリ）。
