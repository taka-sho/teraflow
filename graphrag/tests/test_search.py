"""Tests for search module."""

import json
from unittest.mock import MagicMock, patch

import networkx as nx

from teraflow_graphrag.query.engine import GraphEngine, SearchResult
from teraflow_graphrag.query.search import GlobalSearch, LocalSearch
from teraflow_graphrag.query.summarizer import CommunitySummarizer


def _make_mock_llm(content: str):
    msg = MagicMock()
    msg.content = content
    choice = MagicMock()
    choice.message = msg
    resp = MagicMock()
    resp.choices = [choice]
    return resp


def _make_graph() -> nx.DiGraph:
    g = nx.DiGraph()
    g.add_node("doc-auth", node_type="Document", label="認証設計書")
    g.add_node("auth-svc", node_type="Entity", label="認証サービス")
    g.add_node("jwt-token", node_type="Concept", label="JWTトークン")
    g.add_edge("doc-auth", "auth-svc", edge_type="MENTIONS")
    g.add_edge("auth-svc", "jwt-token", edge_type="RELATED_TO")
    return g


def test_local_search_returns_result():
    graph = _make_graph()
    searcher = LocalSearch(graph, model="claude-haiku-4-5-20251001")
    with patch("litellm.completion", return_value=_make_mock_llm("認証サービスはJWTを使用します。")):
        result = searcher.search("認証")
    assert isinstance(result, SearchResult)
    assert result.mode == "local"
    assert result.query == "認証"
    assert len(result.answer) > 0


def test_local_search_no_match():
    graph = _make_graph()
    searcher = LocalSearch(graph)
    with patch("litellm.completion", return_value=_make_mock_llm("見つかりません。")):
        result = searcher.search("存在しないキーワード12345")
    assert isinstance(result, SearchResult)


def test_local_search_llm_failure():
    graph = _make_graph()
    searcher = LocalSearch(graph)
    with patch("litellm.completion", side_effect=Exception("API error")):
        result = searcher.search("認証")
    assert isinstance(result, SearchResult)
    assert len(result.answer) > 0


def test_global_search_no_summaries(tmp_path):
    searcher = GlobalSearch(output_dir=tmp_path)
    result = searcher.search("認証")
    assert "graph build" in result.answer


def test_global_search_with_summaries(tmp_path):
    summaries_dir = tmp_path / "summaries"
    summaries_dir.mkdir()
    summaries = {
        "0": {"summary": "認証・セキュリティ関連のコミュニティ", "themes": ["認証", "JWT"]},
        "1": {"summary": "ユーザー管理コミュニティ", "themes": ["ユーザー"]},
    }
    (summaries_dir / "communities.json").write_text(json.dumps(summaries, ensure_ascii=False))
    searcher = GlobalSearch(output_dir=tmp_path)
    with patch("litellm.completion", return_value=_make_mock_llm("認証はJWTを使用します。")):
        result = searcher.search("認証")
    assert result.mode == "global"
    assert len(result.sources) > 0


def test_community_summarizer(tmp_path):
    summarizer = CommunitySummarizer()
    nodes = [
        {"node_id": "doc-auth", "label": "認証設計書", "node_type": "Document"},
        {"node_id": "auth-svc", "label": "認証サービス", "node_type": "Entity"},
    ]
    mock_response = json.dumps({"summary": "認証コミュニティ", "themes": ["認証"]})
    with patch("litellm.completion", return_value=_make_mock_llm(mock_response)):
        result = summarizer.summarize_community(0, nodes)
    assert result["community_id"] == 0
    assert result["summary"] == "認証コミュニティ"
    assert result["node_count"] == 2


def test_community_summarizer_failure(tmp_path):
    summarizer = CommunitySummarizer()
    nodes = [{"node_id": "doc-a", "label": "テスト", "node_type": "Document"}]
    with patch("litellm.completion", side_effect=Exception("API error")):
        result = summarizer.summarize_community(0, nodes)
    assert result["summary"] == ""
    assert result["node_count"] == 1


def test_graph_engine_is_abstract():
    import inspect

    assert inspect.isabstract(GraphEngine)
