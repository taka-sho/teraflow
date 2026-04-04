# GitHub Projects 連携設定ガイド

teraflow Phase2 では GitHub Projects（Project V2）のカンバンボードと連携し、
フェーズ遷移やラベル付与に応じて Issue を自動的にボード上で移動できます。

本ドキュメントは PM/PMO 向けの設定手順書です。

---

## 1. 概要

### teraflow × GitHub Projects 連携アーキテクチャ

```
PM/PMO がブラウザで操作
    │
    ├── Issue 作成（フェーズ開始テンプレート）
    │     └→ TF-003 (phase-transition.yml)
    │           ├→ teraflow phase start → state 更新
    │           └→ GraphQL API → Projects ボードで "In Progress" に移動
    │
    ├── ラベル付与（rework-request 等）
    │     └→ TF-012 (rework-impact.yml)
    │           ├→ teraflow rework create → 影響分析
    │           └→ GraphQL API → Projects ボードで "Backlog" に移動
    │
    └── Issue クローズ（フェーズ完了報告）
          └→ TF-003 (phase-transition.yml)
                ├→ teraflow phase complete → state 更新
                └→ GraphQL API → Projects ボードで "Done" に移動

PM/PMO は Projects ボードで全体進捗を俯瞰
  ┌──────────┬──────────────┬──────────┬──────────┬──────────┐
  │ Backlog  │ In Progress  │ Review   │ Done     │ Archived │
  ├──────────┼──────────────┼──────────┼──────────┼──────────┤
  │ Issue#10 │ Issue#5      │ Issue#3  │ Issue#1  │          │
  │          │ Issue#6      │          │ Issue#2  │          │
  └──────────┴──────────────┴──────────┴──────────┴──────────┘
```

### ステージ/フェーズとカラム対応表

| teraflow の状態 | Projects カラム | 移動トリガー |
|----------------|----------------|-------------|
| フェーズ開始前 | **Backlog** | Issue 作成時（デフォルト位置） |
| フェーズ進行中 | **In Progress** | `phase-transition` ラベル付与 / フェーズ開始 Issue 作成 |
| レビュー中 | **Review** | `confirmed` ラベル付与（成果物確定） |
| フェーズ完了 | **Done** | Issue クローズ（フェーズ完了報告） |
| 手戻り発生 | **Backlog** | `rework-*` ラベル付与 |
| 障害対応中 | **In Progress** | `incident-*` ラベル付与 |
| アーカイブ | **Archived** | 全フェーズ完了後に手動移動 |

---

## 2. 前提条件

以下が完了していることを確認してください:

- [ ] teraflow v1.0.0 以上がインストールされている
- [ ] `teraflow setup actions` が完了し、`.github/workflows/teraflow-*.yml` が存在する
- [ ] `teraflow setup templates` が完了し、Issue テンプレートが存在する
- [ ] GitHub リポジトリで Projects (Project V2) が有効になっている
- [ ] `gh` CLI がインストールされ、認証済みである（`gh auth status` で確認）

---

## 3. GitHub Projects カンバンボード設定手順

### 3.1 Project V2 の新規作成

1. GitHub リポジトリのページを開く
2. 上部タブの **Projects** をクリック
3. **New project** → **Board** を選択
4. プロジェクト名を入力（例: `teraflow - Development Pipeline`）
5. **Create project** をクリック

### 3.2 カラム（Status フィールド）の設定

デフォルトで「Todo / In Progress / Done」が作成されます。
以下の手順でカラムを teraflow 向けに変更します:

1. ボード上部の **＋** ボタン、または既存カラム名をクリックして編集
2. 以下の5カラムに変更:

| カラム名 | 色 | 用途 |
|---------|-----|------|
| **Backlog** | グレー | 未着手・手戻り発生 |
| **In Progress** | 青 | フェーズ進行中 |
| **Review** | 黄 | 成果物レビュー・確定待ち |
| **Done** | 緑 | フェーズ完了 |
| **Archived** | 紫 | アーカイブ済み |

3. 不要なデフォルトカラムは **×** で削除

### 3.3 カスタムフィールドの追加（推奨）

| フィールド名 | 型 | 用途 |
|------------|-----|------|
| Group | Single select | グループ名（auth, api, frontend 等） |
| Phase | Single select | 現在フェーズ（requirements, basic_design, detailed_design, implementation, test） |
| Priority | Single select | 優先度（P0, P1, P2） |

設定手順:
1. ボード右上の **＋** をクリック → **New field**
2. フィールド名と型を入力
3. 選択肢を追加

---

## 4. 自動移動ルール設定

### 4.1 GitHub Projects 組み込みワークフロー

GitHub Projects には組み込みの自動化ワークフローがあります:

1. Projects ボード右上の **⚡ Workflows** をクリック
2. 以下のワークフローを有効化:

| ワークフロー | トリガー | アクション |
|------------|---------|----------|
| Item added to project | Issue/PR がプロジェクトに追加 | → **Backlog** に設定 |
| Item closed | Issue がクローズ | → **Done** に設定 |
| Item reopened | Issue が再オープン | → **In Progress** に設定 |
| Pull request merged | PR がマージ | → **Done** に設定 |

### 4.2 teraflow Actions との連携

teraflow のワークフロー（TF-003 等）から GitHub Projects の Issue を自動移動するには、
GraphQL API を使用します。

#### ステップ1: プロジェクト ID とフィールド ID を取得

```bash
# プロジェクト ID を取得（OWNER と NUMBER を置き換え）
gh api graphql -f query='
  query($owner: String!, $number: Int!) {
    user(login: $owner) {
      projectV2(number: $number) {
        id
      }
    }
  }' -f owner="OWNER" -F number=1

# Organization の場合:
gh api graphql -f query='
  query($org: String!, $number: Int!) {
    organization(login: $org) {
      projectV2(number: $number) {
        id
      }
    }
  }' -f org="ORG_NAME" -F number=1
```

#### ステップ2: Status フィールドの ID と選択肢 ID を取得

```bash
# PROJECT_ID を実際の ID に置き換え
gh api graphql -f query='
  query($projectId: ID!) {
    node(id: $projectId) {
      ... on ProjectV2 {
        fields(first: 20) {
          nodes {
            ... on ProjectV2SingleSelectField {
              id
              name
              options {
                id
                name
              }
            }
          }
        }
      }
    }
  }' -f projectId="PROJECT_ID"
```

出力例:
```json
{
  "id": "PVTSSF_xxx",
  "name": "Status",
  "options": [
    { "id": "abc123", "name": "Backlog" },
    { "id": "def456", "name": "In Progress" },
    { "id": "ghi789", "name": "Review" },
    { "id": "jkl012", "name": "Done" },
    { "id": "mno345", "name": "Archived" }
  ]
}
```

#### ステップ3: Issue の ProjectItem ID を取得

```bash
# Issue 番号からプロジェクト内の Item ID を取得
gh api graphql -f query='
  query($owner: String!, $repo: String!, $issue: Int!) {
    repository(owner: $owner, name: $repo) {
      issue(number: $issue) {
        projectItems(first: 10) {
          nodes {
            id
            project { id }
          }
        }
      }
    }
  }' -f owner="OWNER" -f repo="REPO" -F issue=123
```

#### ステップ4: Status を更新（Issue を移動）

```bash
gh api graphql -f query='
  mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
    updateProjectV2ItemFieldValue(
      input: {
        projectId: $projectId
        itemId: $itemId
        fieldId: $fieldId
        value: { singleSelectOptionId: $optionId }
      }
    ) {
      projectV2Item { id }
    }
  }' -f projectId="PROJECT_ID" \
     -f itemId="ITEM_ID" \
     -f fieldId="STATUS_FIELD_ID" \
     -f optionId="TARGET_OPTION_ID"
```

---

## 5. ワークフロー設定への追加手順

既存の teraflow Actions ワークフローに GitHub Projects 連携ステップを追加する方法です。

### 5.1 環境変数の設定

リポジトリの Settings → Secrets and variables → Actions に以下を設定:

| 変数名 | 種別 | 値 |
|--------|------|-----|
| `PROJECT_NUMBER` | Variable | Projects のプロジェクト番号（URL の末尾の数字） |

> **注意**: `GITHUB_TOKEN` は既に Actions で利用可能です。ただし Projects への書き込みには
> `project` スコープの権限が必要です。Organization の Projects を操作する場合は
> Personal Access Token (Classic) を Secrets に設定してください。

### 5.2 teraflow-phase-transition.yml への追加

TF-003 (phase-transition.yml) にGitHub Projects 連携ステップを追加する例:

```yaml
      # === 既存のteraflow実行ステップの後に追加 ===

      - name: Move issue in GitHub Projects
        if: success()
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          OWNER="${{ github.repository_owner }}"
          REPO="${{ github.event.repository.name }}"
          ISSUE_NUMBER="${{ github.event.issue.number }}"
          PROJECT_NUMBER="${{ vars.PROJECT_NUMBER }}"

          if [ -z "$PROJECT_NUMBER" ]; then
            echo "PROJECT_NUMBER not set, skipping Projects integration"
            exit 0
          fi

          # プロジェクト ID 取得
          PROJECT_ID=$(gh api graphql -f query='
            query($owner: String!, $number: Int!) {
              user(login: $owner) {
                projectV2(number: $number) { id }
              }
            }' -f owner="$OWNER" -F number="$PROJECT_NUMBER" \
            --jq '.data.user.projectV2.id' 2>/dev/null || echo "")

          # Organization の場合のフォールバック
          if [ -z "$PROJECT_ID" ]; then
            PROJECT_ID=$(gh api graphql -f query='
              query($org: String!, $number: Int!) {
                organization(login: $org) {
                  projectV2(number: $number) { id }
                }
              }' -f org="$OWNER" -F number="$PROJECT_NUMBER" \
              --jq '.data.organization.projectV2.id' 2>/dev/null || echo "")
          fi

          if [ -z "$PROJECT_ID" ]; then
            echo "Could not find project, skipping"
            exit 0
          fi

          # Issue の ProjectItem ID を取得
          ITEM_ID=$(gh api graphql -f query='
            query($owner: String!, $repo: String!, $issue: Int!) {
              repository(owner: $owner, name: $repo) {
                issue(number: $issue) {
                  projectItems(first: 10) {
                    nodes { id project { id } }
                  }
                }
              }
            }' -f owner="$OWNER" -f repo="$REPO" -F issue="$ISSUE_NUMBER" \
            --jq ".data.repository.issue.projectItems.nodes[] | select(.project.id == \"$PROJECT_ID\") | .id" \
            2>/dev/null || echo "")

          if [ -z "$ITEM_ID" ]; then
            echo "Issue not in project, skipping"
            exit 0
          fi

          # Status フィールド ID と目標カラムの Option ID を取得
          FIELD_DATA=$(gh api graphql -f query='
            query($projectId: ID!) {
              node(id: $projectId) {
                ... on ProjectV2 {
                  fields(first: 20) {
                    nodes {
                      ... on ProjectV2SingleSelectField {
                        id
                        name
                        options { id name }
                      }
                    }
                  }
                }
              }
            }' -f projectId="$PROJECT_ID")

          STATUS_FIELD_ID=$(echo "$FIELD_DATA" | jq -r '.data.node.fields.nodes[] | select(.name == "Status") | .id')

          # ラベルに応じた目標カラムを決定
          TARGET_COLUMN="In Progress"
          EVENT_ACTION="${{ github.event.action }}"
          LABELS="${{ join(github.event.issue.labels.*.name, ',') }}"

          if [ "$EVENT_ACTION" = "closed" ]; then
            TARGET_COLUMN="Done"
          elif echo "$LABELS" | grep -q "rework-"; then
            TARGET_COLUMN="Backlog"
          elif echo "$LABELS" | grep -q "confirmed"; then
            TARGET_COLUMN="Review"
          fi

          OPTION_ID=$(echo "$FIELD_DATA" | jq -r --arg col "$TARGET_COLUMN" \
            '.data.node.fields.nodes[] | select(.name == "Status") | .options[] | select(.name == $col) | .id')

          if [ -n "$STATUS_FIELD_ID" ] && [ -n "$OPTION_ID" ]; then
            gh api graphql -f query='
              mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
                updateProjectV2ItemFieldValue(
                  input: {
                    projectId: $projectId
                    itemId: $itemId
                    fieldId: $fieldId
                    value: { singleSelectOptionId: $optionId }
                  }
                ) {
                  projectV2Item { id }
                }
              }' -f projectId="$PROJECT_ID" \
                 -f itemId="$ITEM_ID" \
                 -f fieldId="$STATUS_FIELD_ID" \
                 -f optionId="$OPTION_ID"

            echo "Moved issue #$ISSUE_NUMBER to '$TARGET_COLUMN'"
          else
            echo "Could not determine field/option IDs, skipping"
          fi
```

---

## 6. ラベル体系と Projects 連携早見表

| teraflow イベント | 発火ワークフロー | ラベル | Projects アクション |
|------------------|----------------|-------|-------------------|
| フェーズ開始 Issue 作成 | TF-003 | `phase-transition` | → **In Progress** |
| フェーズ完了 Issue クローズ | TF-003 | `phase-complete` | → **Done** |
| フェーズスキップ申請 | TF-003 | `phase-skip` | → **Done** |
| 成果物確定ラベル付与 | TF-005 | `confirmed` | → **Review** |
| 手戻りラベル付与 | TF-012 | `rework-request` | → **Backlog** |
| 手戻り承認 | TF-012 | `rework-approved` | → **In Progress** |
| インシデントラベル付与 | TF-014 | `incident-report` | → **In Progress** |
| インシデント解決 | TF-014 | `incident-resolved` | → **Done** |

---

## 7. 設定チェックリスト

### 初期設定（1回のみ）

- [ ] GitHub Projects でプロジェクトを新規作成した
- [ ] カラムを Backlog / In Progress / Review / Done / Archived に設定した
- [ ] リポジトリの Actions Variables に `PROJECT_NUMBER` を設定した
- [ ] （Organization の場合）Projects 操作用の PAT を Secrets に設定した

### teraflow 設定

- [ ] `teraflow setup actions` を実行した
- [ ] `teraflow setup templates` を実行した
- [ ] `teraflow setup labels` を実行した（50+ ラベルが作成済み）

### 動作確認

- [ ] Issue テンプレート「フェーズ開始申請」で Issue を作成した
- [ ] Actions が正常に実行され、Issue が "In Progress" に移動した
- [ ] Issue をクローズし、"Done" に移動することを確認した
- [ ] 手戻りラベルを付与し、"Backlog" に移動することを確認した

### 運用開始

- [ ] PM/PMO メンバーに Projects ボードの URL を共有した
- [ ] カスタムフィールド（Group, Phase）を必要に応じて追加した
- [ ] ボードのフィルタビューを作成した（例: グループ別フィルタ）
