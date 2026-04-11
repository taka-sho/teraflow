"""Tests for graph consistency checks."""
from __future__ import annotations

import networkx as nx

from teraflow_graphrag.graph.schema import EdgeType
from teraflow_graphrag.query.check import check_graph


def _base_graph() -> nx.DiGraph:
    g = nx.DiGraph()
    g.add_node("doc-a", node_type="Document", label="Doc A", status="confirmed")
    g.add_node("doc-b", node_type="Document", label="Doc B", status="confirmed")
    return g


def test_check_valid_graph():
    graph = _base_graph()
    graph.add_edge("doc-a", "doc-b", edge_type=EdgeType.DEPENDS_ON.value)

    result = check_graph(graph)

    assert result.ok is True
    assert result.issues == []


def test_check_broken_ref():
    graph = _base_graph()
    graph.add_node("missing-doc", node_type="Document", label="missing-doc")  # no status
    graph.add_edge("doc-a", "missing-doc", edge_type=EdgeType.DEPENDS_ON.value)

    result = check_graph(graph)

    broken = [i for i in result.issues if i.issue_type == "BrokenRef"]
    assert len(broken) == 1
    assert broken[0].severity == "error"


def test_check_status_conflict():
    graph = _base_graph()
    graph.nodes["doc-a"]["status"] = "confirmed"
    graph.nodes["doc-b"]["status"] = "draft"
    graph.add_edge("doc-a", "doc-b", edge_type=EdgeType.DEPENDS_ON.value)

    result = check_graph(graph)

    conflicts = [i for i in result.issues if i.issue_type == "StatusConflict"]
    assert len(conflicts) == 1
    assert conflicts[0].severity == "warning"


def test_check_implicit_status_conflict():
    graph = _base_graph()
    graph.add_node("entity-x", node_type="Entity", label="Entity X", status="draft")
    graph.add_edge("doc-a", "entity-x", edge_type=EdgeType.MENTIONS.value)

    result = check_graph(graph)

    issues = [i for i in result.issues if i.issue_type == "ImplicitStatusConflict"]
    assert len(issues) == 1
    assert issues[0].source == "graphrag"


def test_check_cyclic_dependency():
    graph = nx.DiGraph()
    graph.add_node("a", node_type="Document", label="A", status="confirmed")
    graph.add_node("b", node_type="Document", label="B", status="confirmed")
    graph.add_node("c", node_type="Document", label="C", status="confirmed")
    graph.add_edge("a", "b", edge_type=EdgeType.DEPENDS_ON.value)
    graph.add_edge("b", "c", edge_type=EdgeType.DEPENDS_ON.value)
    graph.add_edge("c", "a", edge_type=EdgeType.DEPENDS_ON.value)

    result = check_graph(graph)

    cycles = [i for i in result.issues if i.issue_type == "CyclicDependency"]
    assert len(cycles) == 1
    assert cycles[0].severity == "error"


def test_check_to_dict():
    graph = _base_graph()
    graph.add_node("missing-doc", node_type="Document", label="missing-doc")
    graph.add_edge("doc-a", "missing-doc", edge_type=EdgeType.DEPENDS_ON.value)

    result = check_graph(graph)
    payload = result.to_dict()

    assert payload["ok"] is False
    assert payload["issue_count"] >= 1
    assert isinstance(payload["issues"], list)
    assert all("source" in issue for issue in payload["issues"])


def test_check_source_codd_on_broken_ref():
    graph = _base_graph()
    graph.add_node("missing-doc", node_type="Document", label="missing-doc")
    graph.add_edge("doc-a", "missing-doc", edge_type=EdgeType.DEPENDS_ON.value)

    result = check_graph(graph)

    broken = [i for i in result.issues if i.issue_type == "BrokenRef"]
    assert len(broken) == 1
    assert broken[0].source == "codd"
