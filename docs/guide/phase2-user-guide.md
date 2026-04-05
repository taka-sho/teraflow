# Phase2 ユーザーガイド（PM/PMO向け）

## 1. はじめに
Phase2 では、GitHub の Issue/Discussion/Actions を使って、フェーズ管理・手戻り対応・インシデント対応をブラウザだけで運用できます。  
CLI で実行していた一部操作が、Issue テンプレートとラベル操作で自動化されます。  
GitHub Pages 連携により、進捗ダッシュボードを常時確認できます。

Phase1 は主に CLI 中心でしたが、Phase2 は PM/PMO が GitHub UI だけで運用できるように設計されています。

## 2. セットアップ（エンジニアが1回だけ実施）
この章はエンジニア担当です。PM/PMO は実施不要です。

### 2.1 GitHub Actions テンプレート展開
```bash
teraflow setup actions
```

### 2.2 Issue/Discussion テンプレート展開
```bash
teraflow setup templates
```

### 2.3 GitHub Actions Secrets（任意）
リポジトリの `Settings > Secrets and variables > Actions` で `ANTHROPIC_API_KEY` を追加します。  
未設定でも基本機能は動作しますが、AI系ワークフロー（レビュー補助、要件整理など）が制限されます。

### 2.4 GitHub Pages 有効化
1. リポジトリの `Settings > Pages` を開く  
2. `Build and deployment` の Source を `GitHub Actions` に設定  
3. `teraflow-dashboard-deploy` 実行後に Pages URL が払い出されることを確認

## 3. 日常操作ガイド（PM/PMO向け）

### 3.1 フェーズ開始
1. `Issues > New issue` を開く  
2. テンプレート `フェーズ開始申請` を選択  
3. フェーズ名、対象グループ、背景/目的を入力して作成

主な自動処理:
- TF-003 `teraflow-phase-transition` が起動
- `.github/project-state.yml` が更新され、フェーズ遷移が記録される
- Issue に処理結果コメントが返される

### 3.2 フェーズ完了
1. `Issues > New issue` を開く  
2. テンプレート `フェーズ完了申請` を選択  
3. 成果物リンクと完了条件を入力して作成

主な自動処理:
- TF-003 で完了処理
- 条件が揃っていればフェーズ状態が完了へ更新

### 3.3 手戻り起票
1. `Issues > New issue` を開く  
2. テンプレート `手戻り依頼` を選択  
3. 影響範囲、原因、修正方針を入力

`rework-*` ラベルの意味:
- `rework-request`: 手戻り発生の起点
- `rework-approved`: 手戻り承認済み
- `rework-*` が付与されると TF-012 `teraflow-rework-impact` が影響分析を実行

### 3.4 インシデント報告
1. `Issues > New issue` を開く  
2. テンプレート `インシデント報告` を選択  
3. 発生日時、影響範囲、暫定対応を入力

`incident-*` ラベルの使い方:
- `incident-critical`: 緊急度が最も高い障害
- `incident-high`: 高優先度障害
- `incident-*` ラベルで TF-014 `teraflow-incident-agent` が調査コメントを返す

### 3.5 AI機能の活用
- PRレビュー自動化: `ANTHROPIC_API_KEY` 設定済みの場合、TF-009 が PR にレビューコメントを投稿
- 要件整理: Discussions の要件議論カテゴリで TF-004 が論点整理を支援
- CI修正提案: CI失敗時に TF-008 が修正方針を提案

## 4. ラベル一覧
| ラベル | 意味 | 自動実行 |
|--------|------|---------|
| 確定 / confirmed | 成果物確定 | TF-005: changelog更新 |
| rework-* | 手戻り発生 | TF-012: 影響分析 |
| conflict | コンフリクト | TF-010: AI解消提案 |
| incident-* | インシデント | TF-014: AI調査 |
| agent-implement | AI実装依頼 | TF-007: AI実装提案 |
| stuck-monitor | 停滞アラート | 自動生成 |

## 5. ダッシュボード確認
1. `Settings > Pages` で公開 URL を確認  
2. ダッシュボードで以下を確認:
- 現在フェーズとステータス
- 手戻り/インシデント件数の推移
- 直近の自動処理結果（更新日時）

## 6. よくある質問 (FAQ)
### Q1. Actions が動かない
A. `Settings > Actions` で Actions が有効か確認し、workflow 実行権限が制限されていないか確認してください。

### Q2. AI機能が動かない
A. `Settings > Secrets and variables > Actions` に `ANTHROPIC_API_KEY` が設定されているか確認してください。

### Q3. ラベルがない
A. エンジニアに `teraflow label setup` の再実行を依頼してください。
