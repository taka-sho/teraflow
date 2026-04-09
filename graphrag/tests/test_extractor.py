"""Tests for entity extractor (using mock LLM responses)."""

import json
from unittest.mock import MagicMock, patch

from teraflow_graphrag.extract.extractor import EntityExtractor, ExtractionResult
from teraflow_graphrag.extract.prompts import build_extraction_prompt
from teraflow_graphrag.graph.schema import EdgeType, NodeType


MOCK_RESPONSE = json.dumps(
    {
        "entities": [
            {
                "entity_id": "auth-service",
                "label": "認証サービス",
                "entity_type": "Entity",
                "description": "ユーザー認証を担当するサービス",
            },
            {
                "entity_id": "jwt-token",
                "label": "JWTトークン",
                "entity_type": "Concept",
                "description": "認証トークン形式",
            },
        ],
        "relations": [
            {
                "source_id": "doc-001",
                "target_id": "auth-service",
                "relation_type": "MENTIONS",
                "description": "文書が認証サービスに言及",
            },
            {
                "source_id": "auth-service",
                "target_id": "jwt-token",
                "relation_type": "RELATED_TO",
                "description": "認証サービスはJWTを使用",
            },
        ],
    }
)


def _make_mock_response(content: str):
    msg = MagicMock()
    msg.content = content
    choice = MagicMock()
    choice.message = msg
    response = MagicMock()
    response.choices = [choice]
    return response


def test_extract_entities_and_relations():
    extractor = EntityExtractor(model="claude-haiku-4-5-20251001")
    with patch("litellm.completion", return_value=_make_mock_response(MOCK_RESPONSE)):
        result = extractor.extract(
            node_id="doc-001",
            title="認証設計書",
            content="認証サービスはJWTトークンを使用する。",
        )
    assert len(result.entities) == 2
    assert result.entities[0].node_id == "auth-service"
    assert result.entities[0].node_type == NodeType.ENTITY
    assert result.entities[1].node_type == NodeType.CONCEPT
    assert len(result.relations) == 2
    assert result.relations[0].edge_type == EdgeType.MENTIONS
    assert result.relations[1].edge_type == EdgeType.RELATED_TO


def test_extract_invalid_json():
    extractor = EntityExtractor()
    with patch("litellm.completion", return_value=_make_mock_response("not json")):
        result = extractor.extract(node_id="doc-001", title="test", content="test")
    assert len(result.entities) == 0
    assert len(result.relations) == 0


def test_extract_llm_failure():
    extractor = EntityExtractor()
    with patch("litellm.completion", side_effect=Exception("API error")):
        result = extractor.extract(node_id="doc-001", title="test", content="test")
    assert isinstance(result, ExtractionResult)
    assert len(result.entities) == 0


def test_extract_empty_entities():
    response = json.dumps({"entities": [], "relations": []})
    extractor = EntityExtractor()
    with patch("litellm.completion", return_value=_make_mock_response(response)):
        result = extractor.extract(node_id="doc-001", title="test", content="test")
    assert result.entities == []
    assert result.relations == []


def test_build_extraction_prompt_contains_known_entities():
    known = [{"entity_id": "auth-service", "label": "認証サービス"}]
    prompt = build_extraction_prompt(
        node_id="doc-001",
        title="テスト",
        status="draft",
        content="テスト内容",
        known_entities=known,
    )
    assert "auth-service" in prompt
    assert "doc-001" in prompt


def test_build_extraction_prompt_no_known_entities():
    prompt = build_extraction_prompt(
        node_id="doc-001",
        title="テスト",
        status="confirmed",
        content="テスト内容",
        known_entities=[],
    )
    assert "（なし）" in prompt
