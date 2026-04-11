"""Tests for impact analysis module."""
from __future__ import annotations

import networkx as nx

from teraflow_graphrag.graph.schema import EdgeType
from teraflow_graphrag.query.impact import analyze_impact


def _make_graph() -> nx.DiGraph:
    g = nx.DiGraph()
    g.add_node("doc-a", node_type="Document", label="Doc A", status="confirmed")
    g.add_node("doc-b", node_type="Document", label="Doc B", status="review")
    g.add_node("doc-c", node_type="Document", label="Doc C", status="draft")
    g.add_node("entity-x", node_type="Entity", label="Entity X", status="draft")

    g.add_edge("doc-a", "doc-b", edge_type=EdgeType.DEPENDS_ON.value)
    g.add_edge("doc-b", "doc-c", edge_type=EdgeType.DEPENDS_ON.value)
    g.add_edge("entity-x", "doc-c", edge_type=EdgeType.MENTIONS.value)
    return g


def test_impact_direct_dependents():
    graph = _make_graph()
    result = analyze_impact(graph, "doc-b", depth=2)
    affected_ids = {n.node_id for n in result.affected_nodes}
    assert "doc-a" in affected_ids


def test_impact_depth_limit():
    graph = _make_graph()
    result = analyze_impact(graph, "doc-c", depth=1)
    affected_ids = {n.node_id for n in result.affected_nodes}
    assert "doc-a" not in affected_ids


def test_impact_depth_2():
    graph = _make_graph()
    result = analyze_impact(graph, "doc-c", depth=2)
    affected_ids = {n.node_id for n in result.affected_nodes}
    assert "doc-a" in affected_ids


def test_impact_source_codd():
    graph = _make_graph()
    result = analyze_impact(graph, "doc-b", depth=2)
    doc_a = next(n for n in result.affected_nodes if n.node_id == "doc-a")
    assert doc_a.source == "codd"


def test_impact_codd_only_mode():
    graph = _make_graph()
    result = analyze_impact(graph, "doc-c", depth=2, include_graphrag=False)
    affected_ids = {n.node_id for n in result.affected_nodes}
    assert "entity-x" not in affected_ids


def test_impact_nonexistent_node():
    graph = _make_graph()
    result = analyze_impact(graph, "missing-node", depth=2)
    assert result.total_count == 0
    assert result.affected_nodes == []


def test_impact_to_dict():
    graph = _make_graph()
    result = analyze_impact(graph, "doc-b", depth=2)
    payload = result.to_dict()
    assert payload["root_node_id"] == "doc-b"
    assert payload["total_count"] >= 1
    assert isinstance(payload["affected_nodes"], list)
    assert all("source" in item for item in payload["affected_nodes"])
