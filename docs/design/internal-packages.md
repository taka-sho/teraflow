---
codd:
  node_id: "design:internal-packages"
  title: "内部パッケージ設計（state, config）"
  depends_on:
    - id: "design:cli-interface"
      relation: derives_from
    - id: "adr:002-cli-framework"
      relation: derives_from
---

# 内部パッケージ設計（state, config）

## 概要

Phase1の共通パッケージとして `internal/state` と `internal/config` を設計する。
現行の `cmd/init.go` ではYAML生成がインライン文字列（`renderTeraflowYAML`, `renderProjectStateYAML`）で実装されている。これを構造体ベースの型安全なYAML R/Wに置き換え、stage/phase/reworkコマンドから共有可能にする。

## 1. internal/state パッケージ

### 役割

`.github/project-state.yml` の読み書きと `.teraflow/rework-log.yml` の管理。

### 型定義

```go
package state

import "time"

// ProjectState は .github/project-state.yml のルート構造。
type ProjectState struct {
    Project   ProjectInfo   `yaml:"project"`
    Lifecycle Lifecycle     `yaml:"lifecycle"`
    Phases    PhaseState    `yaml:"phases"`
}

type ProjectInfo struct {
    Name string `yaml:"name"`
}

type Lifecycle struct {
    CurrentStage string `yaml:"current_stage"`
}

type PhaseState struct {
    Current string `yaml:"current"`
}

// ReworkEntry は手戻り1件の記録。
type ReworkEntry struct {
    ID          string    `yaml:"id"`
    Group       string    `yaml:"group"`
    TargetPhase string    `yaml:"target_phase"`
    Reason      string    `yaml:"reason"`
    CreatedAt   time.Time `yaml:"created_at"`
    Status      string    `yaml:"status"` // open | approved | rejected
}

// ReworkLog は .teraflow/rework-log.yml のルート構造。
type ReworkLog struct {
    Reworks []ReworkEntry `yaml:"reworks"`
}
```

### 公開関数

```go
// LoadState は configPath（.github/teraflow.yml のパス）から
// 同ディレクトリの project-state.yml を探してロードする。
//
// configPath が ".github/teraflow.yml" なら
// ".github/project-state.yml" を読み込む。
func LoadState(configPath string) (*ProjectState, error)

// SaveState は ProjectState を project-state.yml に書き戻す。
// configPath から保存先パスを導出する。
func SaveState(configPath string, s *ProjectState) error

// LoadReworkLog は configPath から .teraflow/rework-log.yml を探してロードする。
// ファイルが存在しない場合は空の ReworkLog を返す（エラーではない）。
//
// パス導出: configPath ".github/teraflow.yml"
// → プロジェクトルート（.github/ の親）
// → .teraflow/rework-log.yml
func LoadReworkLog(configPath string) (*ReworkLog, error)

// AppendRework は rework-log.yml に1エントリを追記する。
// ファイルが存在しない場合は新規作成する。
func AppendRework(configPath string, entry ReworkEntry) error
```

### パス導出ロジック

```go
// resolveProjectRoot は configPath から .github/ を取り除き、
// プロジェクトルートを返す。
//
// ".github/teraflow.yml"         → "."
// "/abs/path/.github/teraflow.yml" → "/abs/path"
func resolveProjectRoot(configPath string) string {
    return filepath.Dir(filepath.Dir(configPath))
}

// stateFilePath は configPath と同じディレクトリの project-state.yml を返す。
func stateFilePath(configPath string) string {
    return filepath.Join(filepath.Dir(configPath), "project-state.yml")
}

// reworkLogPath はプロジェクトルートの .teraflow/rework-log.yml を返す。
func reworkLogPath(configPath string) string {
    root := resolveProjectRoot(configPath)
    return filepath.Join(root, ".teraflow", "rework-log.yml")
}
```

### LoadState 実装方針

```go
func LoadState(configPath string) (*ProjectState, error) {
    path := stateFilePath(configPath)
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to load project state: %w", err)
    }
    var s ProjectState
    if err := yaml.Unmarshal(data, &s); err != nil {
        return nil, fmt.Errorf("failed to parse project state: %w", err)
    }
    return &s, nil
}
```

### SaveState 実装方針

```go
func SaveState(configPath string, s *ProjectState) error {
    path := stateFilePath(configPath)
    data, err := yaml.Marshal(s)
    if err != nil {
        return fmt.Errorf("failed to marshal project state: %w", err)
    }
    return os.WriteFile(path, data, 0o644)
}
```

### AppendRework 実装方針

```go
func AppendRework(configPath string, entry ReworkEntry) error {
    log, err := LoadReworkLog(configPath)
    if err != nil {
        return err
    }
    log.Reworks = append(log.Reworks, entry)

    path := reworkLogPath(configPath)
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return err
    }
    data, err := yaml.Marshal(log)
    if err != nil {
        return fmt.Errorf("failed to marshal rework log: %w", err)
    }
    return os.WriteFile(path, data, 0o644)
}
```

---

## 2. internal/config パッケージ

### 役割

`.github/teraflow.yml` の読み書きと設定値の取得・設定。

### 型定義

```go
package config

// TeraflowConfig は .github/teraflow.yml のルート構造。
type TeraflowConfig struct {
    Version      string        `yaml:"version"`
    Project      ProjectConfig `yaml:"project"`
    Confirmation Confirmation  `yaml:"confirmation"`
    AI           AIConfig      `yaml:"ai"`
    Harness      Harness       `yaml:"harness"`
}

type ProjectConfig struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
    Repository  string `yaml:"repository"`
}

type Confirmation struct {
    Trigger    string `yaml:"trigger"`
    ReqTrigger string `yaml:"req_trigger"`
}

type AIConfig struct {
    DefaultProvider string `yaml:"default_provider"`
}

type Harness struct {
    ScoreThreshold int  `yaml:"score_threshold"`
    AutoIssue      bool `yaml:"auto_issue"`
}
```

### ConfigKey定数

```go
// ConfigKey は config set/get で扱えるキーを定義する。
type ConfigKey string

const (
    KeyVersion          ConfigKey = "version"
    KeyProjectName      ConfigKey = "project.name"
    KeyProjectDesc      ConfigKey = "project.description"
    KeyProjectRepo      ConfigKey = "project.repository"
    KeyConfirmTrigger   ConfigKey = "confirmation.trigger"
    KeyConfirmReqTrigger ConfigKey = "confirmation.req_trigger"
    KeyAIProvider       ConfigKey = "ai.default_provider"
    KeyHarnessThreshold ConfigKey = "harness.score_threshold"
    KeyHarnessAutoIssue ConfigKey = "harness.auto_issue"
)

// AllKeys は config set/get で使用可能なキー一覧を返す。
func AllKeys() []ConfigKey {
    return []ConfigKey{
        KeyVersion, KeyProjectName, KeyProjectDesc, KeyProjectRepo,
        KeyConfirmTrigger, KeyConfirmReqTrigger,
        KeyAIProvider, KeyHarnessThreshold, KeyHarnessAutoIssue,
    }
}
```

### 公開関数

```go
// Load は指定パスの teraflow.yml を読み込む。
// path は --config フラグの値（デフォルト: ".github/teraflow.yml"）。
func Load(path string) (*TeraflowConfig, error)

// Save は TeraflowConfig を指定パスに書き戻す。
func Save(path string, cfg *TeraflowConfig) error

// GetValue は ConfigKey に対応する値を文字列で返す。
// 未知のキーの場合はエラーを返す。
func GetValue(cfg *TeraflowConfig, key string) (string, error)

// SetValue は ConfigKey に対応するフィールドに値を設定する。
// int/bool型フィールドは文字列からパースする。
// 未知のキーの場合はエラーを返す。
func SetValue(cfg *TeraflowConfig, key, value string) error
```

### GetValue / SetValue 実装方針

```go
func GetValue(cfg *TeraflowConfig, key string) (string, error) {
    switch ConfigKey(key) {
    case KeyProjectName:
        return cfg.Project.Name, nil
    case KeyProjectDesc:
        return cfg.Project.Description, nil
    case KeyProjectRepo:
        return cfg.Project.Repository, nil
    case KeyAIProvider:
        return cfg.AI.DefaultProvider, nil
    case KeyHarnessThreshold:
        return strconv.Itoa(cfg.Harness.ScoreThreshold), nil
    case KeyHarnessAutoIssue:
        return strconv.FormatBool(cfg.Harness.AutoIssue), nil
    // ... 他のキー
    default:
        return "", fmt.Errorf("unknown config key: %s", key)
    }
}

func SetValue(cfg *TeraflowConfig, key, value string) error {
    switch ConfigKey(key) {
    case KeyProjectName:
        cfg.Project.Name = value
    case KeyHarnessThreshold:
        v, err := strconv.Atoi(value)
        if err != nil {
            return fmt.Errorf("invalid value for %s: %w", key, err)
        }
        cfg.Harness.ScoreThreshold = v
    case KeyHarnessAutoIssue:
        v, err := strconv.ParseBool(value)
        if err != nil {
            return fmt.Errorf("invalid value for %s: %w", key, err)
        }
        cfg.Harness.AutoIssue = v
    // ... 他のキー
    default:
        return fmt.Errorf("unknown config key: %s", key)
    }
    return nil
}
```

---

## 3. rework-log.yml スキーマ設計

### ファイルパス

`.teraflow/rework-log.yml`

`.github/` ではなく `.teraflow/` 配下に配置する理由:
- rework-logは運用データ（頻繁に追記される）
- `.github/` はGitHub設定領域。運用データを混在させない
- 変更ログ（`.teraflow/changelog/`）と同じ領域に統一

### スキーマ

```yaml
# .teraflow/rework-log.yml
reworks:
  - id: "rw-001"
    group: "group-auth"
    target_phase: "basic_design"
    reason: "API仕様変更により基本設計の見直しが必要"
    created_at: "2026-04-10T10:00:00Z"
    status: "open"        # open | approved | rejected

  - id: "rw-002"
    group: "group-api"
    target_phase: "requirement_definition"
    reason: "要件の追加により要件定義フェーズに差し戻し"
    created_at: "2026-04-12T14:30:00Z"
    status: "approved"
    approved_at: "2026-04-12T15:00:00Z"
    approved_by: "user-a"
```

### ID採番ルール

```
rw-{連番3桁}
```

`AppendRework` は既存ログの最大IDを取得し、+1したIDを付与する。ログが空の場合は `rw-001` から開始。

```go
func nextReworkID(log *ReworkLog) string {
    maxNum := 0
    for _, rw := range log.Reworks {
        // "rw-001" → 1
        if n, err := strconv.Atoi(strings.TrimPrefix(rw.ID, "rw-")); err == nil && n > maxNum {
            maxNum = n
        }
    }
    return fmt.Sprintf("rw-%03d", maxNum+1)
}
```

---

## 4. テスト方針

### internal/state テスト

```go
// state_test.go

func TestLoadState(t *testing.T) {
    // tempdir に .github/project-state.yml を配置
    // LoadState で読み込み、フィールド値を検証
}

func TestSaveState(t *testing.T) {
    // tempdir に保存後、再読み込みして一致を検証
}

func TestLoadState_FileNotFound(t *testing.T) {
    // 存在しないパスでエラーを返すことを検証
}

func TestLoadReworkLog_EmptyFile(t *testing.T) {
    // ファイルが存在しない場合、空の ReworkLog を返すことを検証
}

func TestLoadReworkLog_ExistingEntries(t *testing.T) {
    // 既存エントリを持つrework-log.ymlの読み込み検証
}

func TestAppendRework(t *testing.T) {
    // 空ファイルへの追記 → 1件のエントリが記録される
    // 既存ファイルへの追記 → エントリが末尾に追加される
}

func TestAppendRework_IDAutoIncrement(t *testing.T) {
    // rw-001 が存在する場合、次は rw-002 が付与される
}

func TestResolveProjectRoot(t *testing.T) {
    // ".github/teraflow.yml" → "."
    // "/abs/path/.github/teraflow.yml" → "/abs/path"
}
```

### internal/config テスト

```go
// config_test.go

func TestLoad(t *testing.T) {
    // tempdir に teraflow.yml を配置
    // Load で読み込み、全フィールドを検証
}

func TestSave(t *testing.T) {
    // 保存後に再読み込みして一致を検証
}

func TestLoad_FileNotFound(t *testing.T) {
    // 存在しないパスでエラーを返すことを検証
}

func TestGetValue(t *testing.T) {
    // 各 ConfigKey について正しい値が返ることを検証
}

func TestGetValue_UnknownKey(t *testing.T) {
    // 未知のキーでエラーを返すことを検証
}

func TestSetValue(t *testing.T) {
    // string型キー: 設定後に GetValue で一致を検証
    // int型キー（harness.score_threshold）: "80" → 80 の変換を検証
    // bool型キー（harness.auto_issue）: "true" → true の変換を検証
}

func TestSetValue_InvalidType(t *testing.T) {
    // int型キーに "abc" → エラーを検証
    // bool型キーに "maybe" → エラーを検証
}
```

### テスト用ヘルパー

```go
// testdata に置くのではなく、テスト内で動的に生成する。
// 理由: YAML内容をテストケースごとに変えたいため。

func setupTestProject(t *testing.T) (configPath string, cleanup func()) {
    t.Helper()
    dir := t.TempDir()
    githubDir := filepath.Join(dir, ".github")
    os.MkdirAll(githubDir, 0o755)

    // teraflow.yml
    cfg := renderTeraflowYAML("test-project")
    os.WriteFile(filepath.Join(githubDir, "teraflow.yml"), []byte(cfg), 0o644)

    // project-state.yml
    state := renderProjectStateYAML("test-project", "initial_development")
    os.WriteFile(filepath.Join(githubDir, "project-state.yml"), []byte(state), 0o644)

    return filepath.Join(githubDir, "teraflow.yml"), func() {}
    // t.TempDir() は自動クリーンアップ
}
```

---

## 5. cmd/init.go リファクタリング方針

現行の `cmd/init.go` は `renderTeraflowYAML` / `renderProjectStateYAML` で文字列テンプレートを使用している。internal/state, internal/config 導入後のリファクタリング方針:

```go
// Before (現行):
func renderTeraflowYAML(name string) string {
    return fmt.Sprintf(`version: "1"
project:
  name: %q
...`, name)
}

// After (リファクタリング後):
func runInit(opts initOptions, cwd string) error {
    // config パッケージで構造体を生成し、YAML書き出し
    cfg := &config.TeraflowConfig{
        Version: "1",
        Project: config.ProjectConfig{
            Name: name,
        },
        Confirmation: config.Confirmation{
            Trigger:    "確定",
            ReqTrigger: "要求確定",
        },
        AI: config.AIConfig{
            DefaultProvider: "anthropic",
        },
        Harness: config.Harness{
            ScoreThreshold: 70,
            AutoIssue:      false,
        },
    }
    if err := config.Save(cfgPath, cfg); err != nil {
        return err
    }

    // state パッケージで構造体を生成し、YAML書き出し
    s := &state.ProjectState{
        Project:   state.ProjectInfo{Name: name},
        Lifecycle: state.Lifecycle{CurrentStage: opts.Stage},
        Phases:    state.PhaseState{Current: "requirements"},
    }
    if err := state.SaveState(cfgPath, s); err != nil {
        return err
    }

    // ... 他のファイル生成
}
```

このリファクタリングにより:
- 型安全なYAML生成（フィールド漏れをコンパイル時に検出）
- state/config パッケージを他コマンド（stage/phase/rework）から再利用可能
- テスト容易性の向上（構造体の比較テスト）

---

## 6. パッケージ間の依存関係

```
cmd/init.go ─────uses────▶ internal/config
                 ─────uses────▶ internal/state

cmd/stage/*.go ──uses────▶ internal/state  （Phase1: stage advance/current/activate/freeze）
cmd/phase/*.go ──uses────▶ internal/state  （Phase1: phase advance/current/skip/status）
cmd/rework/*.go ─uses────▶ internal/state  （Phase1: rework create/approve → AppendRework）

internal/state ──imports──▶ gopkg.in/yaml.v3, os, path/filepath, fmt, time, strconv, strings
internal/config ─imports──▶ gopkg.in/yaml.v3, os, fmt, strconv
```

state と config は互いに依存しない。cmd/ 層が両方を使用する。
