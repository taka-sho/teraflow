"""Tests for community detection module."""
from __future__ import annotations

from unittest.mock import patch

import networkx as nx

from teraflow_graphrag.graph.community import (
    _fallback_components,
    _load_communities,
    _save_communities,
    add_community_nodes,
    detect_communities,
)
from teraflow_graphrag.graph.schema import NodeType
from teraflow_graphrag.storage.state import GraphState


def _make_graph() -> nx.DiGraph:
    g = nx.DiGraph()
    g.add_nodes_from(["doc-a", "doc-b", "doc-c", "doc-d"])
    g.add_edges_from([("doc-a", "doc-b"), ("doc-c", "doc-d")])
    return g


def test_detect_communities_skips_when_below_threshold(tmp_path):
    existing = {"doc-a": 0, "doc-b": 0}
    _save_communities(existing, tmp_path)

    graph = _make_graph()
    state = GraphState(total_edge_count=10, community_edge_count=10)

    result = detect_communities(graph, state, tmp_path, threshold=0.1)
    assert result == existing


def test_detect_communities_runs_when_above_threshold(tmp_path):
    graph = _make_graph()
    state = GraphState(total_edge_count=11, community_edge_count=10)

    mock_partition = {"doc-a": 0, "doc-b": 0, "doc-c": 1, "doc-d": 1}
    with patch(
        "teraflow_graphrag.graph.community._run_leiden", return_value=mock_partition
    ):
        result = detect_communities(graph, state, tmp_path, threshold=0.05)

    assert result == mock_partition
    assert state.community_edge_count == graph.number_of_edges()


def test_detect_communities_force(tmp_path):
    graph = _make_graph()
    state = GraphState(total_edge_count=2, community_edge_count=2)

    mock_partition = {"doc-a": 0, "doc-b": 0, "doc-c": 1, "doc-d": 1}
    with patch(
        "teraflow_graphrag.graph.community._run_leiden", return_value=mock_partition
    ):
        result = detect_communities(graph, state, tmp_path, threshold=0.1, force=True)

    assert len(result) == 4


def test_fallback_components():
    graph = _make_graph()
    result = _fallback_components(graph)
    assert result["doc-a"] == result["doc-b"]
    assert result["doc-c"] == result["doc-d"]
    assert result["doc-a"] != result["doc-c"]


def test_fallback_on_import_error(tmp_path):
    graph = _make_graph()
    state = GraphState(total_edge_count=3, community_edge_count=1)

    with patch.dict("sys.modules", {"graspologic": None, "graspologic.partition": None}):
        result = detect_communities(graph, state, tmp_path, threshold=0.1)

    assert len(result) == 4


def test_add_community_nodes():
    graph = nx.DiGraph()
    graph.add_nodes_from(["doc-a", "doc-b", "doc-c"])
    communities = {"doc-a": 0, "doc-b": 0, "doc-c": 1}

    updated = add_community_nodes(graph, communities)

    assert updated.has_node("community-0")
    assert updated.has_node("community-1")
    assert updated.has_edge("doc-a", "community-0")
    assert updated.has_edge("doc-c", "community-1")
    assert updated.nodes["community-0"]["node_type"] == NodeType.COMMUNITY.value


def test_save_and_load_communities(tmp_path):
    communities = {"doc-a": 0, "doc-b": 0, "doc-c": 1}
    _save_communities(communities, tmp_path)
    loaded = _load_communities(tmp_path)
    assert loaded == communities


def test_load_communities_missing(tmp_path):
    result = _load_communities(tmp_path)
    assert result == {}
