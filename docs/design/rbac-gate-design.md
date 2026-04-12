---
node_id: "design:rbac-gate-design"
depends_on: []
tags: ["design", "rbac", "gate", "access-control"]
status: approved
---

# RBAC・Gate・Constraint・Audit 設計書

Wave2実装の基盤設計。REQ-SLCP-005〜008を実現する。

## A. RBAC設計

### A-1. Permission型

```go
// internal/rbac/rbac.go

// Permission represents a single permission check.
type Permission struct {
    Resource string // "gate", "process", "stage", "phase", "rework", "audit"
    Action   string // "approve", "start", "complete", "advance", "list", "read"
    Target   string // process name or "*" for wildcard
}

// String returns "resource.action.target" format.
func (p Permission) String() string {
    return p.Resource + "." + p.Action + "." + p.Target
}

// Match checks if this permission satisfies the required permission.
// Supports wildcards: "gate.approve.*" matches "gate.approve.detailed_design"
func (p Permission) Match(required Permission) bool {
    if p.Resource != required.Resource && p.Resource != "*" {
        return false
    }
    if p.Action != required.Action && p.Action != "*" {
        return false
    }
    if p.Target != required.Target && p.Target != "*" {
        return false
    }
    return true
}
```

### A-2. Role型

```go
// Role represents a named role with a set of permissions.
type Role struct {
    Name        string       `yaml:"name"`
    Members     []MemberRef  `yaml:"members"`
    Permissions []string     `yaml:"permissions"` // "gate.approve.*", "process.start", etc.
}

// MemberRef identifies a user by github username or team.
type MemberRef struct {
    GitHub string `yaml:"github,omitempty"` // GitHub username
    Team   string `yaml:"team,omitempty"`   // GitHub Team name (Phase2)
}

// HasPermission checks if this role grants the given permission.
func (r Role) HasPermission(required Permission) bool {
    for _, perm := range r.Permissions {
        p := ParsePermission(perm)
        if p.Match(required) {
            return true
        }
    }
    return false
}
```

### A-3. RBACConfig (teraflow.yml rbacセクション)

```yaml
# .github/teraflow.yml
rbac:
  enabled: true           # false(デフォルト) = 全操作許可（後方互換）

  roles:
    pm:
      members:
        - github: "taka-sho"
      permissions:
        - "gate.approve.*"
        - "process.start"
        - "process.complete"
        - "stage.advance"
        - "constraint.override"    # --force 使用権限
        - "*.read"

    architect:
      members:
        - github: "architect-user"
      permissions:
        - "gate.approve.basic_design"
        - "gate.approve.detailed_design"
        - "process.start"
        - "process.complete"
        - "*.read"

    developer:
      members:
        - github: "dev-user"
      permissions:
        - "process.start"
        - "rework.create"
        - "incident.create"
        - "*.read"

    qa:
      members:
        - github: "qa-user"
      permissions:
        - "gate.approve.testing"
        - "gate.approve.integration_test"
        - "incident.create"
        - "incident.close"
        - "*.read"

    release_mgr:
      members:
        - github: "release-user"
      permissions:
        - "gate.approve.release"
        - "stage.advance"
        - "constraint.override"
        - "*.read"
```

```go
// internal/rbac/config.go

// RBACConfig represents the rbac section of teraflow.yml.
type RBACConfig struct {
    Enabled bool            `yaml:"enabled"`
    Roles   map[string]Role `yaml:"roles"`
}
```

### A-4. CheckPermission関数

```go
// CheckPermission checks if a user has the required permission.
// Returns nil if allowed, error if denied.
// If RBAC is disabled (config.Enabled == false), always returns nil.
func CheckPermission(config *RBACConfig, user string, required Permission) error {
    if config == nil || !config.Enabled {
        return nil // RBAC disabled = all allowed
    }

    role := ResolveRole(config, user)
    if role == nil {
        return fmt.Errorf("permission denied: user %q has no role assigned in rbac config", user)
    }

    if role.HasPermission(required) {
        return nil
    }

    return fmt.Errorf("permission denied: role %q does not have %s (user: %s)",
        role.Name, required.String(), user)
}

// ResolveRole finds the role for a given user.
// Checks github field in each role's members list.
func ResolveRole(config *RBACConfig, user string) *Role {
    for _, role := range config.Roles {
        for _, member := range role.Members {
            if member.GitHub == user {
                return &role
            }
        }
    }
    return nil
}

// GetCurrentUser returns the current user from git config.
func GetCurrentUser() (string, error) {
    // exec: git config user.name
    // fallback: os.Getenv("USER")
}
```

### A-5. ユーザーロール設定

```bash
# ユーザーが自分のロールを設定（Phase1簡易認証）
teraflow config set role pm

# 設定は .teraflow/user-config.yml に保存
# user:
#   role: pm
#   name: taka-sho  # git config user.name から自動取得
```

ロール解決の優先順位:
1. teraflow.yml rbac.roles.*.members でユーザー名マッチ
2. .teraflow/user-config.yml の role 設定
3. マッチなし → RBACエラー（rbac.enabled=true時）

## B. Gate設計

### B-1. GateRule型

```go
// internal/gate/gate.go

// GateRule defines the completion conditions for a process.
type GateRule struct {
    ProcessName string      `yaml:"process_name"`
    Conditions  []Condition `yaml:"conditions"`
}

// Condition represents a single gate condition.
type Condition struct {
    Type      string  `yaml:"type"`       // document_exists, manual_approval, test_pass_rate, coverage
    Path      string  `yaml:"path,omitempty"`       // for document_exists
    Threshold float64 `yaml:"threshold,omitempty"`   // for test_pass_rate, coverage (0.0〜1.0)
}

// ConditionResult represents the evaluation result of one condition.
type ConditionResult struct {
    Type      string  `yaml:"type"`
    Satisfied bool    `yaml:"satisfied"`
    Current   string  `yaml:"current"`   // "0.85" for coverage, "exists" for document_exists
    Required  string  `yaml:"required"`  // "0.80" for threshold, "docs/design/" for path
    Message   string  `yaml:"message"`
}

// EvaluateGate checks all conditions for a process.
// Returns results for each condition and overall pass/fail.
func EvaluateGate(rule GateRule, projectRoot string) ([]ConditionResult, bool) {
    results := make([]ConditionResult, 0, len(rule.Conditions))
    allPassed := true
    for _, cond := range rule.Conditions {
        result := evaluateCondition(cond, projectRoot)
        results = append(results, result)
        if !result.Satisfied {
            allPassed = false
        }
    }
    return results, allPassed
}
```

### B-2. teraflow.yml gate_rulesセクション

```yaml
# .github/teraflow.yml
gate_rules:
  requirements:
    conditions:
      - type: document_exists
        path: "docs/requirements/"
      - type: manual_approval
  basic_design:
    conditions:
      - type: document_exists
        path: "docs/design/"
      - type: manual_approval
  detailed_design:
    conditions:
      - type: document_exists
        path: "docs/design/"
      - type: manual_approval
  implementation:
    conditions:
      - type: test_pass_rate
        threshold: 1.0
      - type: coverage
        threshold: 0.70
  testing:
    conditions:
      - type: test_pass_rate
        threshold: 1.0
      - type: coverage
        threshold: 0.80
      - type: manual_approval
```

### B-3. `teraflow gate approve <process_name>` フロー

```
ユーザー: teraflow gate approve detailed_design
  │
  ├─1. GetCurrentUser() → "taka-sho"
  │
  ├─2. CheckPermission(rbac, user, Permission{gate, approve, detailed_design})
  │    ├─ OK → 続行
  │    └─ NG → エラー: "permission denied: role 'developer' does not have gate.approve.detailed_design"
  │
  ├─3. LoadGateRules(configPath) → GateRule for "detailed_design"
  │
  ├─4. EvaluateGate(rule, projectRoot)
  │    ├─ document_exists("docs/design/") → ✅ exists
  │    └─ manual_approval → ❌ (この approve コマンド自体が承認操作)
  │
  ├─5. ゲート条件表示
  │    [✅] ドキュメント存在: docs/design/ (exists)
  │    [→] 手動承認: このコマンドで承認します
  │
  ├─6. SLCPJCFProcess "ソフトウェア設計プロセス"
  │    Status: in_progress → completed
  │    CompletedAt: 2026-04-05T17:00:00
  │
  ├─7. AppendAudit(AuditEntry{
  │      User: "taka-sho", Role: "pm",
  │      Action: "gate.approve", Target: "detailed_design",
  │      Result: "approved"
  │    })
  │
  └─8. SaveState + 出力: "Gate approved: detailed_design ✅"
```

### B-4. Phase1での条件評価ロジック

| 条件タイプ | 評価方法 | 実装 |
|-----------|---------|------|
| `document_exists` | `os.Stat(filepath.Join(root, path))` | S |
| `manual_approval` | `teraflow gate approve` コマンド自体が承認 | S |
| `test_pass_rate` | `go test ./... -json` の結果パース | M |
| `coverage` | `go test -coverprofile` → カバレッジ率算出 | M |

Phase2で追加: `issue_label`, `pr_merged`, `issue_ratio`（GitHub API）

## C. Constraint設計

### C-1. Constraint型

```go
// internal/constraint/constraint.go

// Constraint defines a rule that blocks or warns on certain operations.
type Constraint struct {
    ID        string `yaml:"id"`
    Trigger   string `yaml:"trigger"`    // "stage.advance", "phase.complete", "gate.approve"
    Condition string `yaml:"condition"`  // "rework.open > 0", "coverage < 0.80"
    Action    string `yaml:"action"`     // "block" or "warn"
    Message   string `yaml:"message"`
}

// Violation represents a constraint that was triggered.
type Violation struct {
    Constraint Constraint
    Current    string // 現在値（"open_count=2", "coverage=0.75"等）
}

// Engine evaluates constraints against the current state.
type Engine struct {
    Constraints []Constraint
}

// Check evaluates all constraints matching the trigger.
// Returns violations (block/warn) and whether any are blocking.
func (e *Engine) Check(trigger string, ctx *EvalContext) ([]Violation, bool) {
    var violations []Violation
    hasBlock := false
    for _, c := range e.Constraints {
        if c.Trigger != trigger {
            continue
        }
        if violated := evaluateCondition(c, ctx); violated {
            v := Violation{Constraint: c, Current: ctx.Describe(c.Condition)}
            violations = append(violations, v)
            if c.Action == "block" {
                hasBlock = true
            }
        }
    }
    return violations, hasBlock
}

// EvalContext holds state data for constraint evaluation.
type EvalContext struct {
    ReworkOpenCount    int
    IncidentOpenCount  int
    CriticalIncidents  int
    CoverageRate       float64
    TestPassRate       float64
    DocumentPaths      []string // 存在するドキュメントパス
}
```

### C-2. teraflow.yml constraintsセクション

```yaml
# .github/teraflow.yml
constraints:
  - id: C001
    trigger: "stage.advance"
    condition: "rework.open > 0"
    action: warn
    message: "未解決の手戻りが{{count}}件あります"

  - id: C002
    trigger: "gate.approve.testing"
    condition: "test_pass_rate < 1.0"
    action: block
    message: "テスト通過率が100%未満です（現在: {{rate}}%）"

  - id: C003
    trigger: "gate.approve.testing"
    condition: "coverage < 0.80"
    action: block
    message: "カバレッジが80%未満です（現在: {{rate}}%）"

  - id: C004
    trigger: "stage.advance"
    condition: "incident.critical > 0"
    action: block
    message: "未解決のクリティカル障害があります"
```

### C-3. block vs warn 動作仕様

```
block:
  ユーザー: teraflow stage advance
  システム: ❌ BLOCKED: テスト通過率が100%未満です（現在: 95%）[C002]
           操作を中止しました。
           → テストを修正して再実行してください
           → 緊急時: teraflow stage advance --force --reason="理由"

warn:
  ユーザー: teraflow stage advance
  システム: ⚠️ WARNING: 未解決の手戻りが2件あります [C001]
           続行しますか？ (y/N): y
           → Advanced: release
```

### C-4. --force --reason フラグ

```go
// cmd/stage.go (advance コマンド拡張)

var forceFlag bool
var reasonFlag string

// フラグ定義
cmd.Flags().BoolVar(&forceFlag, "force", false, "Override constraint blocks")
cmd.Flags().StringVar(&reasonFlag, "reason", "", "Reason for force override (required with --force)")

// 制約チェック
violations, hasBlock := constraintEngine.Check("stage.advance", evalCtx)
if hasBlock && !forceFlag {
    // block違反を表示してエラー終了
    return fmt.Errorf("blocked by constraint %s", violations[0].Constraint.ID)
}
if hasBlock && forceFlag {
    // --force 権限チェック
    if err := rbac.CheckPermission(config, user, Permission{"constraint", "override", "*"}); err != nil {
        return fmt.Errorf("--force requires pm or release_mgr role: %w", err)
    }
    if reasonFlag == "" {
        return errors.New("--force requires --reason flag")
    }
    // 監査ログに記録
    audit.Append(AuditEntry{
        Action: "constraint.override",
        Target: "stage.advance",
        Result: "forced",
        Note:   reasonFlag,
        ConstraintsOverridden: violationIDs(violations),
    })
}
```

## D. Audit設計

### D-1. AuditEntry型

```go
// internal/audit/audit.go

// AuditEntry represents a single audit log record.
type AuditEntry struct {
    ID                    string   `yaml:"id"`
    Timestamp             string   `yaml:"timestamp"`
    User                  string   `yaml:"user"`
    Role                  string   `yaml:"role"`
    Action                string   `yaml:"action"`  // "gate.approve", "stage.advance", "constraint.override"
    Target                string   `yaml:"target"`  // "detailed_design", "initial_development → release"
    Result                string   `yaml:"result"`  // "approved", "denied", "forced", "warned"
    ConstraintsOverridden []string `yaml:"constraints_overridden,omitempty"`
    Note                  string   `yaml:"note,omitempty"`
}

// AuditLog represents .teraflow/audit-log.yml.
type AuditLog struct {
    Entries []AuditEntry `yaml:"entries"`
}

// AppendAudit appends an entry to .teraflow/audit-log.yml.
func AppendAudit(configPath string, entry AuditEntry) error {
    entry.ID = generateAuditID()     // "AUD-001" 形式
    entry.Timestamp = time.Now().Format(time.RFC3339)
    // ... load, append, save
}

// LoadAuditLog loads .teraflow/audit-log.yml.
// Returns empty log if file does not exist.
func LoadAuditLog(configPath string) (*AuditLog, error) {
    // ... similar to LoadReworkLog pattern
}
```

### D-2. audit-log.yml の場所

```
.teraflow/
  audit-log.yml    ← 監査ログ（新規）
  rework-log.yml   ← 既存
  incident-log.yml ← 既存
  user-config.yml  ← ユーザーロール設定（新規）
```

### D-3. `teraflow audit list` コマンド

```go
// cmd/audit.go

func newAuditCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "audit",
        Short: "View audit log",
    }
    cmd.AddCommand(newAuditListCmd())
    return cmd
}

func newAuditListCmd() *cobra.Command {
    var userFilter, actionFilter, sinceFilter string

    cmd := &cobra.Command{
        Use:   "list",
        Short: "List audit log entries",
        // フィルタ適用 → テーブル表示 or JSON出力
    }

    cmd.Flags().StringVar(&userFilter, "user", "", "Filter by user")
    cmd.Flags().StringVar(&actionFilter, "action", "", "Filter by action")
    cmd.Flags().StringVar(&sinceFilter, "since", "", "Filter entries after date (YYYY-MM-DD)")
    return cmd
}
```

### D-4. 監査ログ記録タイミング

| トリガー | Action | 記録内容 |
|---------|--------|---------|
| `teraflow gate approve <name>` | `gate.approve` | 承認者、対象プロセス、条件充足状況 |
| `teraflow stage advance` | `stage.advance` | 遷移元→遷移先、制約違反の有無 |
| `teraflow phase complete` | `phase.complete` | 完了フェーズ、次フェーズ |
| `teraflow process start <name>` | `process.start` | 開始プロセス |
| `teraflow process complete` | `process.complete` | 完了プロセス |
| RBAC拒否 | `permission.denied` | 拒否されたユーザー、要求権限、現在ロール |
| `--force` 使用 | `constraint.override` | オーバーライドした制約ID、理由 |

## E. Process状態マシン図

```
                    process start <name>           gate approve <name>
                    (権限: process.start)           (権限: gate.approve.<name>)
                         │                              │
    ┌─────────────┐      │      ┌──────────────┐        │       ┌───────────────┐
    │ not_started  │─────────→│  in_progress  │──────────────→│   completed    │
    └─────────────┘              └──────────────┘                └───────────────┘
          │                           │                                │
          │                           │                                │
          └── 初期状態                 ├── StartedAt 設定               ├── CompletedAt 設定
              (DefaultSLCPJCF         ├── CurrentProcess 更新          ├── gate_passed = true
               Processes で作成)      └── audit記録                    ├── 次プロセスの CurrentProcess 更新
                                                                      └── audit記録
```

### 遷移ルール

| 遷移 | トリガーコマンド | RBAC権限 | 前提条件 |
|------|----------------|----------|---------|
| not_started → in_progress | `teraflow process start <name>` | `process.start` | 前プロセスがcompletedであること |
| in_progress → completed | `teraflow gate approve <name>` | `gate.approve.<name>` | GateRule条件を全て充足 |

### phase/stage連動

```
teraflow phase complete
  → 対応するSLCPJCFProcessがin_progressなら、gate approve を促すメッセージ表示
  → gate_rulesが未設定（or 全条件充足済み）なら自動でcompleted

teraflow stage advance
  → constraintチェック実行
  → 全プロセスの状態をリセット（次ステージ用）
```

## F. ラベル統一方針

### 現状の不一致

| # | state.go (DefaultSLCPJCFProcesses) | process.go (slcpJCFProcessLabels) |
|---|------------------------------------|------------------------------------|
| 1 | 企画プロセス | 企画 |
| 2 | 要件定義プロセス | 要件定義 |
| 3 | システム設計プロセス | システム設計 |
| 4 | ソフトウェア設計プロセス | ソフトウェア設計 |
| 5 | ソフトウェア構築プロセス | 構築 |
| 6 | ソフトウェアテストプロセス | テスト |
| 7 | システム結合テスト | リリース |
| 8 | 運用・保守プロセス | 運用保守 |

### 統一案: 正式名称 + 短縮ラベル

state.goの正式名称をマスターデータとし、process.goは表示用の短縮ラベルを別途定義する。

```go
// internal/state/state.go — マスター定義（変更なし）
// Name フィールドは正式名称のまま維持

// cmd/process.go — 表示用マッピング追加
var processDisplayLabels = map[string]string{
    "企画プロセス":             "企画",
    "要件定義プロセス":         "要件定義",
    "システム設計プロセス":     "システム設計",
    "ソフトウェア設計プロセス": "ソフトウェア設計",
    "ソフトウェア構築プロセス": "構築",
    "ソフトウェアテストプロセス": "テスト",
    "システム結合テスト":       "結合テスト",
    "運用・保守プロセス":       "運用保守",
}

// CLIコマンドのキー（英語）
var processKeys = map[string]string{
    "企画プロセス":             "planning",
    "要件定義プロセス":         "requirements",
    "システム設計プロセス":     "system_design",
    "ソフトウェア設計プロセス": "software_design",
    "ソフトウェア構築プロセス": "implementation",
    "ソフトウェアテストプロセス": "testing",
    "システム結合テスト":       "integration_test",
    "運用・保守プロセス":       "operation_maintenance",
}
```

CLIコマンドでは英語キーを使用:
```bash
teraflow process start planning
teraflow gate approve detailed_design
```

表示では日本語短縮ラベルを使用:
```
企画 ✅ → 要件定義 ✅ → システム設計 🔄 → ...
```

### #7 の不一致修正

state.goの「システム結合テスト」は共通フレームの2.3.5-6に相当。process.goの「リリース」は2.3.7-8に相当。概念が異なるため、state.goの名称を修正:

```go
// 修正前: {Name: "システム結合テスト", Status: "not_started"}
// 修正後: state.goのまま維持し、process.goのラベルをstate.goに合わせる
// "リリース" → "結合テスト" に変更
```

## G. 新規パッケージ・ファイル一覧

| パッケージ/ファイル | 種別 | 内容 |
|-------------------|------|------|
| `internal/rbac/rbac.go` | 新規 | Permission, Role, CheckPermission, ResolveRole |
| `internal/rbac/config.go` | 新規 | RBACConfig, LoadRBACConfig |
| `internal/rbac/rbac_test.go` | 新規 | ユニットテスト |
| `internal/gate/gate.go` | 新規 | GateRule, Condition, EvaluateGate |
| `internal/gate/gate_test.go` | 新規 | ユニットテスト |
| `internal/constraint/constraint.go` | 新規 | Constraint, Engine, Check |
| `internal/constraint/constraint_test.go` | 新規 | ユニットテスト |
| `internal/audit/audit.go` | 新規 | AuditEntry, AppendAudit, LoadAuditLog |
| `internal/audit/audit_test.go` | 新規 | ユニットテスト |
| `cmd/gate.go` | 新規 | teraflow gate approve コマンド |
| `cmd/gate_test.go` | 新規 | ユニットテスト |
| `cmd/audit.go` | 新規 | teraflow audit list コマンド |
| `cmd/audit_test.go` | 新規 | ユニットテスト |
| `cmd/process.go` | 改修 | start/complete サブコマンド追加、ラベル統一 |
| `cmd/stage.go` | 改修 | constraint + RBAC チェック注入 |
| `cmd/phase.go` | 改修 | process連動 |
| `cmd/status.go` | 改修 | 動的ロール別表示 |
| `cmd/config.go` | 改修 | set role サブコマンド追加 |
