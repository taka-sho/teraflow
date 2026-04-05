# RBAC（Role-Based Access Control）ガイド

## 概要

teraflow の RBAC は、ユーザーごとに実行可能な操作を制御する仕組みです。`teraflow.yml` の `rbac` セクションでロール・メンバー・権限を定義し、CLI と GitHub Actions から同じルールを利用します。

主な用途:
- `gate approve` / `rbac apply` などの操作権限を制限する
- GitHub Issue ベースのロール申請を自動処理する
- 監査ログ（`audit.log`）と組み合わせて変更履歴を追跡する

## セットアップ

### teraflow.yml 設定

最小構成例:

```yaml
rbac:
  enabled: true
  admin_role: admin
  github_enforcement: true
  roles:
    - name: admin
      members: ["taka-sho"]
      permissions: ["*"]
    - name: pm
      members: ["pm-user1"]
      permissions:
        - "gate.approve.*"
        - "stage.advance"
        - "phase.complete"
        - "rbac.apply"
    - name: dev
      members: ["dev-user1"]
      permissions:
        - "phase.complete"
```

設定ポイント:
- `enabled: true` で RBAC を有効化
- `github_enforcement: true` の場合、CLI 実行時に `--user` が必須になる操作がある
- `permissions` は `*` とワイルドカード（例: `gate.approve.*`）を利用可能

### admin_role の設定

`admin_role` を明示しない場合は `admin` が既定値です。
`admin_role` は最終管理者保護に利用され、最後の admin メンバーを削除する変更は拒否されます。

## ロール管理

### Issueテンプレートによるロール申請

`role-management.yml` テンプレートで以下を申請します:
- 操作（add / change / remove）
- 対象ユーザー
- 変更先ロール
- 変更元ロール（change のみ）
- 変更理由

Issue に `rbac` ラベルが付くと、`teraflow-rbac` ワークフローが起動します。

### teraflow rbac apply コマンド

Issue 申請内容を `teraflow.yml` に反映します。

```bash
teraflow rbac apply --from-issue 123 --user taka-sho --config .github/teraflow.yml
```

主な挙動:
- `--from-issue` で GitHub Issue 本文を取得
- 操作内容を解析してロールメンバーを更新
- `rbac.apply` 権限がないユーザーは拒否

## 権限チェック

### teraflow rbac check

指定ユーザーに操作権限があるかを判定します。

```bash
teraflow rbac check --user alice --action gate.approve.planning --config .github/teraflow.yml
```

JSON 出力例:

```bash
teraflow rbac check \
  --user alice \
  --action gate.approve.planning \
  --format json
```

### GitHub Actions での自動チェック

`teraflow-rbac.yml` では、Issue 作成者の権限を `teraflow rbac check` で検証してから `rbac apply` を実行します。

典型フロー:
1. `rbac` ラベル付き Issue を検知
2. 申請者に `rbac.apply` 権限があるか確認
3. 権限ありなら変更適用と PR 作成
4. 権限なしなら Issue にコメントしてクローズ

## よくある質問（FAQ）

Q. `rbac check` が `--user flag is required` で失敗します。  
A. `github_enforcement: true` のためです。`--user <github-login>` を指定してください。

Q. 最後の admin を remove できません。  
A. 安全のためブロックされます。先に別ユーザーへ admin ロールを付与してください。

Q. `gate approve` が permission denied になります。  
A. 対象プロセスに対応する権限（例: `gate.approve.planning`）が必要です。`rbac list` と `rbac check` で確認してください。
