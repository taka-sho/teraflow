---
codd:
  node_id: "adr:008-phase1-coexistence"
  title: "ADR-008: Phase1 CLI とPhase2の共存・データ整合性"
  depends_on:
    - id: "req:phase2-github-integration"
      relation: implements
    - id: "adr:004-data-github"
      relation: extends
    - id: "adr:006-event-driven"
      relation: refines
---

# ADR-008: Phase1 CLI とPhase2の共存・データ整合性

## ステータス

提案（Proposed）

## コンテキスト

Phase2導入後、Phase1のCLIユーザー（エンジニア）とPhase2のGitHub.comユーザー（PM/PMO）が同一のproject-state.ymlを読み書きする。データ整合性の確保が必要（W-005対応）。

### 競合シナリオ

```
Time  CLI User (Engineer)              Actions (GitHub.com trigger)
─────────────────────────────────────────────────────────────────
T1    git pull (state: phase=design)
T2                                      Issue closed → Actions起動
T3                                      git pull (state: phase=design)
T4    teraflow phase advance            teraflow phase advance
T5    git push (state: phase=impl)      git push → CONFLICT!
```

## 決定

**git commit + pushをSSoTとした楽観的並行制御を採用する。**

### 設計

1. **CLI側**: 通常のgitワークフロー。`git pull` → `teraflow <command>` → `git commit` → `git push`
2. **Actions側**: リトライ付きpush
   ```bash
   MAX_RETRY=3
   for i in $(seq 1 $MAX_RETRY); do
     git pull --rebase origin main
     teraflow <command>
     git add .github/project-state.yml .teraflow/
     git commit -m "teraflow: <operation>"
     if git push origin main; then
       break
     fi
     if [ $i -eq $MAX_RETRY ]; then
       # 通知: Issueコメントで報告
       gh issue comment $ISSUE_NUMBER --body "⚠️ State update failed after $MAX_RETRY retries. Manual intervention required."
       exit 1
     fi
     sleep 5
   done
   ```
3. **concurrencyグループ**: 同一Actions間の並行実行を防止
   ```yaml
   concurrency:
     group: teraflow-state-update
     cancel-in-progress: false
   ```
4. **CLI + Actions同時実行**: concurrencyはActions間のみ有効。CLI操作との競合はgit conflictで検知

### 競合発生時の対応

| ケース | 検知方法 | 対応 |
|--------|---------|------|
| Actions同士 | concurrency group | 逐次実行（競合しない） |
| CLI vs Actions | git push失敗 | Actions: 3回リトライ。CLI: 手動pull+再操作 |
| CLI vs CLI | git push失敗 | 手動pull+再操作（通常の git ワークフロー） |

### 代替案（却下）

| 案 | 却下理由 |
|----|---------|
| ファイルロック（flock） | ローカル限定。リモートCLIとActionsの排他不可 |
| 楽観的ロック（last_updated比較） | YAML内にバージョンフィールド追加が必要。git自体が同等機能を提供 |
| DB（Phase4方式） | Phase2ではオーバーエンジニアリング |

## 影響

- CLIユーザーは操作前に `git pull` を推奨（ドキュメントに明記）
- Actions側の3回リトライで大半の競合は自動解決
- 高頻度の同時操作（10+ users同時フェーズ遷移）は想定外。Phase3（GitHub App + DB）で対応
- changelog（JSONL追記）はconflict しにくい（追記のみ）。project-state.ymlが主な競合対象
