"""Prompts for entity extraction from CoDD documents (design doc ch.9)."""

ENTITY_EXTRACTION_SYSTEM = """あなたは技術文書からエンティティと関係を抽出する専門家です。
CoDD（Concern-Oriented Domain Documentation）文書の依存関係グラフを構築します。"""

ENTITY_EXTRACTION_PROMPT = """以下のCoDD文書からエンティティと関係を抽出してください。

## 文書情報
- node_id: {node_id}
- タイトル: {title}
- ステータス: {status}

## 文書内容
{content}

## 既知のエンティティ（重複抽出防止 - 設計書9章）
以下のエンティティはすでにグラフに存在します。新規エンティティを追加する場合は
既存エンティティと同一概念でないか確認してください:
{known_entities}

## 抽出指示
1. この文書が言及するエンティティ（概念、コンポーネント、技術用語）を最大10件抽出
2. エンティティ間の関係（RELATED_TO）を抽出
3. 既存エンティティと同一概念の場合は既存IDを使用し、新規エンティティを作成しない

## 出力形式（JSON）
{{
  "entities": [
    {{
      "entity_id": "一意のID（英小文字+ハイフン、例: auth-service）",
      "label": "エンティティ名",
      "entity_type": "Entity または Concept",
      "description": "簡潔な説明（1行）"
    }}
  ],
  "relations": [
    {{
      "source_id": "エンティティIDまたはnode_id",
      "target_id": "エンティティIDまたはnode_id",
      "relation_type": "MENTIONS または RELATED_TO",
      "description": "関係の説明"
    }}
  ]
}}

JSONのみを出力してください。説明文は不要です。"""


def build_extraction_prompt(
    node_id: str,
    title: str,
    status: str,
    content: str,
    known_entities: list[dict],
) -> str:
    known_str = (
        "\n".join(f"- {e['entity_id']}: {e['label']}" for e in known_entities[:50])
        if known_entities
        else "（なし）"
    )
    return ENTITY_EXTRACTION_PROMPT.format(
        node_id=node_id,
        title=title,
        status=status or "unknown",
        content=content[:4000],  # token budget guard
        known_entities=known_str,
    )
