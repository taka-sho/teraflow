---
codd:
  node_id: business:marketplace-strategy
  title: "teraflow Marketplace戦略・付加価値機能・料金プラン"
  depends_on:
    - id: "req:teraflow-overview"
      relation: derives_from
    - id: "adr:003-ai-integration"
      relation: derives_from
---

# teraflow Marketplace戦略

## 1. 付加価値機能一覧（V-001〜V-010）

| ID | 機能名 | 概要 | 対応プラン |
|----|--------|------|-----------|
| V-001 | Check Runs（ゲート検証） | PR時にフェーズゲート条件を自動チェック。NGならマージブロック | Pro〜 |
| V-002 | PR自動ラベリング | 変更ファイルのパスからフェーズ・ステージラベルを自動付与 | Free〜 |
| V-003 | 影響分析コメント | PRにcodd impact結果を自動コメント。影響範囲を可視化 | Pro〜 |
| V-004 | ライフサイクルダッシュボード | フェーズ進捗・KPIをWebで可視化 | Pro〜 |
| V-005 | 手戻り検知アラート | 完了フェーズへの変更を検知し手戻りコスト自動算出→Issue起票 | Enterprise |
| V-006 | Discussion→Issue自動変換 | 「確定」ラベル付与時に要件Issueを自動生成 | Pro〜 |
| V-007 | 成果物一貫性監査 | 定期codd scanで依存グラフ不整合を検出→レポート | Enterprise |
| V-008 | マスタースケジュール予測 | コミット速度・Issue消化率から完了日予測・遅延警告 | Enterprise |
| V-009 | CODEOWNERS自動生成 | ロール・権限管理からCODEOWNERS自動生成・更新 | Pro〜 |
| V-010 | 変更ログ自動生成 | マージ時changelog自動追記・リリースノート生成 | Free〜 |

## 2. プラン別機能マッピング

| 機能 | Free | Pro ($9/user/月) | Enterprise ($25/user/月) |
|------|------|-----------------|--------------------------|
| V-001 Check Runs | - | ○ | ○ |
| V-002 PR自動ラベリング | ○ | ○ | ○ |
| V-003 影響分析コメント | - | ○ | ○ |
| V-004 ライフサイクルダッシュボード | - | ○ | ○ |
| V-005 手戻り検知アラート | - | - | ○ |
| V-006 Discussion→Issue変換 | - | ○ | ○ |
| V-007 成果物一貫性監査 | - | - | ○ |
| V-008 スケジュール予測 | - | - | ○ |
| V-009 CODEOWNERS自動生成 | - | ○ | ○ |
| V-010 変更ログ自動生成 | ○ | ○ | ○ |
| マルチリポジトリ対応 | - | - | ○ |
| GitHub Enterprise対応 | - | - | ○ |

### 無料トライアル
Pro/Enterpriseともに14日間無料トライアルを提供。

## 3. 料金体系・手数料（cmd_085調査結果）

| 項目 | 内容 |
|------|------|
| GitHub Marketplace手数料 | 5% |
| Stripe決済手数料 | 3% |
| **実質手数料合計** | **8%** |
| 競合価格帯 | $5〜25/user/月 |

### AI機能（B-003: マルチベンダー）の課金方式
- Pro: プラン内包（月100リクエストまで）
- Enterprise: プラン内包（月500リクエストまで）
- 上限超過: $0.01/リクエストの従量課金

## 4. 競合差別化

### 主な差別化ポイント
- **上流工程カバー**: 要件定義→設計→実装の一貫性管理はMarketplaceにほぼ競合なし
- **CoDD方法論統合**: 依存グラフベースの影響分析（V-003, V-007）は独自機能
- **Terasoluna対応**: 日本の大規模SI向けプロセス管理に特化
- **4Phase拡張性**: CLI（Phase1）→GitHub App（Phase3）→SaaS（Phase4）の段階的進化

### 競合比較（調査対象3件）
| ツール種別 | 価格帯 | teraflowとの差異 |
|-----------|--------|-----------------|
| CI/CD管理系 | $10〜30/seat | 実装フェーズのみ。設計・要件管理なし |
| プロジェクト管理系 | $5〜20/user | GitHub連携が補助的。ライフサイクル管理なし |
| コードレビュー系 | $15〜25/user | 実装後フェーズのみ。上流工程なし |

## 5. Phase4（TeraflowHub SaaS）への発展パス

### Marketplace→SaaSアップセルパス
1. **Phase3**: GitHub Appとして Marketplace掲載（per-seat課金）
2. **Phase4移行トリガー**: 組織全体での利用、外部ステークホルダー連携ニーズ
3. **SaaS移行インセンティブ**: マルチプロジェクト管理、カスタムダッシュボード、SLA保証
4. **課金併存**: Marketplace版（GitHub内）とSaaS版（独自Web UI）を並行提供可能

### TeraflowHub SaaS追加機能（Phase4）
- マルチテナント組織管理
- カスタムAIプロバイダー設定（B-003）
- 外部PMOツール連携（Jira, Confluence等）
- 高度なアナリティクス・レポーティング
